package livesmoke

import (
	"bufio"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// exitWait bounds how long the client waits for the server to exit once its output ended or its
// stdin was closed.
const exitWait = 5 * time.Second

// minWait is the shortest wait for one reply line (Python's queue.get(timeout=max(0.01, ...))).
const minWait = 10 * time.Millisecond

// Client is a minimal newline-delimited JSON-RPC client over a server subprocess.
type Client struct {
	timeout time.Duration
	id      int
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	lines   chan string   // stdout lines; closed at EOF
	exited  chan struct{} // closed once the process was reaped
	code    string        // the exit code as Python prints returncode, once exited
	mu      sync.Mutex
	stderr  []string
}

// StartClient launches argv (in dir when non-empty) with environ; timeout bounds every request.
func StartClient(argv []string, dir string, environ []string, timeout time.Duration) (*Client, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir, cmd.Env = dir, environ
	c := &Client{timeout: timeout, cmd: cmd, lines: make(chan string, 64), exited: make(chan struct{})}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	c.stdin = stdin
	var pumps sync.WaitGroup
	pumps.Add(2)
	go func() { defer pumps.Done(); pump(stdout, func(l string) { c.lines <- l }); close(c.lines) }()
	go func() {
		defer pumps.Done()
		pump(stderr, func(l string) { c.mu.Lock(); c.stderr = append(c.stderr, l); c.mu.Unlock() })
	}()
	go func() { pumps.Wait(); _ = cmd.Wait(); c.code = exitCode(cmd.ProcessState); close(c.exited) }()
	return c, nil
}

// pump hands every line of r (without its newline) to sink, up to EOF.
func pump(r io.Reader, sink func(string)) {
	br := bufio.NewReader(r)
	for {
		line, err := br.ReadString('\n')
		if line != "" {
			sink(strings.TrimSuffix(line, "\n"))
		}
		if err != nil {
			return
		}
	}
}

// StderrLines is what the server wrote on stderr so far, one entry per line.
func (c *Client) StderrLines() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string{}, c.stderr...)
}

// exitedCode is the exit code once the process exited within wait, else "None".
func (c *Client) exitedCode(wait time.Duration) string {
	select {
	case <-c.exited:
		return c.code
	case <-time.After(wait):
		return "None"
	}
}

func (c *Client) send(msg *pyjson.Object) error {
	data, err := pyjson.Marshal(msg, pyjson.Options{})
	if err == nil {
		_, err = c.stdin.Write(append(data, '\n'))
	}
	if err != nil {
		return smokef("server exited (code %s)", c.exitedCode(0))
	}
	return nil
}

// Request sends one JSON-RPC request and returns its result.
func (c *Client) Request(method string, params *pyjson.Object) (any, error) {
	c.id++
	if params == nil {
		params = pyjson.NewObject()
	}
	if err := c.send(kwargs("jsonrpc", "2.0", "id", c.id, "method", method, "params", params)); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(c.timeout)
	for {
		var line string
		var open bool
		wait := time.Until(deadline)
		if wait < minWait { // every wait is at least minWait, even past the deadline
			wait = minWait
		}
		timer := time.NewTimer(wait)
		select {
		case line, open = <-c.lines:
			timer.Stop()
		case <-timer.C:
			return nil, smokef("timeout waiting for %s", method)
		}
		if !open {
			return nil, smokef("server exited (code %s)", c.exitedCode(exitWait))
		}
		reply, err := pyjson.Loads([]byte(line))
		obj, isObj := reply.(*pyjson.Object)
		if err != nil || !isObj {
			continue
		}
		if id, _ := obj.Get("id"); !numberEquals(id, c.id) {
			continue
		}
		if e, failed := obj.Get("error"); failed {
			return nil, smokef("protocol error on %s: %s", method, pyjson.Str(e))
		}
		return item(obj, "result")
	}
}

// Start is the MCP handshake: initialize, then the initialized notification.
func (c *Client) Start() error {
	info := kwargs("name", "live-smoke", "version", "1")
	if _, err := c.Request("initialize", kwargs("protocolVersion", "2025-06-18",
		"capabilities", pyjson.NewObject(), "clientInfo", info)); err != nil {
		return err
	}
	return c.send(kwargs("jsonrpc", "2.0", "method", "notifications/initialized"))
}

// Call is one tools/call: the error text when the result is an error, else its
// structuredContent.
func (c *Client) Call(tool string, args *pyjson.Object) (bool, any, error) {
	res, err := c.Request("tools/call", kwargs("name", tool, "arguments", args))
	if err != nil {
		return false, nil, err
	}
	isErr, err := get(res, "isError")
	if err != nil {
		return false, nil, err
	}
	if pyjson.Truthy(isErr) {
		text, err := errorText(res)
		return true, text, err
	}
	sc, err := item(res, "structuredContent")
	return false, sc, err
}

func errorText(res any) (any, error) {
	content, err := item(res, "content")
	if err != nil {
		return nil, err
	}
	first, err := index(content, 0)
	if err != nil {
		return nil, err
	}
	return item(first, "text")
}

// Close closes the server's stdin and waits for it to exit; past the wait it kills the server
// (only the one this client started).
func (c *Client) Close() {
	_ = c.stdin.Close()
	go func() { // unread output must not block the reaper
		for range c.lines {
		}
	}()
	select {
	case <-c.exited:
	case <-time.After(exitWait):
		_ = c.cmd.Process.Kill()
		<-c.exited
	}
}
