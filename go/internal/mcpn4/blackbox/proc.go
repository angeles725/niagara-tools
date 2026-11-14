package blackbox

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

// Timeout bounds every wait for the server: one reply, or the process exit.
const Timeout = 30 * time.Second

// Proc is one mcp-n4 process with raw access to its stdio: what the shutdown and end-to-end
// tests need. Server builds the request/reply protocol on top of it.
type Proc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	lines  chan string // stdout lines without "\n"; closed at EOF
	stderr syncBuffer
	done   chan struct{}
	code   int
}

// Spawn starts the server with o's flags and environment; the test's cleanup kills it if it
// still runs.
func Spawn(t testing.TB, o Options) *Proc {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	p := &Proc{cmd: exec.Command(exe, o.Argv()...), lines: make(chan string, 64), done: make(chan struct{})}
	p.cmd.Env = append(o.Environ(), marker+"=1")
	p.cmd.Stderr = &p.stderr
	if p.stdin, err = p.cmd.StdinPipe(); err == nil {
		p.stdout, err = p.cmd.StdoutPipe()
	}
	if err == nil {
		err = p.cmd.Start()
	}
	if err != nil {
		t.Fatal(err)
	}
	go p.read()
	t.Cleanup(p.Kill)
	return p
}

func (p *Proc) read() {
	sc := bufio.NewScanner(p.stdout)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		p.lines <- sc.Text()
	}
	close(p.lines)
	err := p.cmd.Wait()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		p.code = exit.ExitCode() // -1 when a signal killed it
	case err != nil:
		p.code = -1
	}
	close(p.done)
}

// Send writes one line (a newline is added) to the server's stdin.
func (p *Proc) Send(line string) error {
	_, err := io.WriteString(p.stdin, line+"\n")
	return err
}

// ReadLine is the next stdout line; ok is false at EOF or after Timeout.
func (p *Proc) ReadLine() (line string, ok bool) {
	select {
	case line, ok = <-p.lines:
		return line, ok
	case <-time.After(Timeout):
		return "", false
	}
}

// CloseStdin closes the server's stdin (EOF ends a serving server).
func (p *Proc) CloseStdin() { _ = p.stdin.Close() }

// CloseStdout closes the read end of the server's stdout: its next reply cannot be written.
func (p *Proc) CloseStdout() { _ = p.stdout.Close() }

// Signal sends sig to the server.
func (p *Proc) Signal(sig os.Signal) error { return p.cmd.Process.Signal(sig) }

// Wait waits for the exit; it returns the exit code (-1 for a signal death), the stdout lines
// not read yet, and the whole stderr. It fails the test after Timeout.
func (p *Proc) Wait(t testing.TB) (code int, rest []string, stderr string) {
	t.Helper()
	for {
		select {
		case line, ok := <-p.lines:
			if ok {
				rest = append(rest, line)
				continue
			}
			select {
			case <-p.done:
				return p.code, rest, p.stderr.String()
			case <-time.After(Timeout):
			}
		case <-time.After(Timeout):
		}
		t.Fatalf("the server did not exit within %s; stderr: %s", Timeout, p.stderr.String())
	}
}

// Stderr is what the server wrote to stderr so far.
func (p *Proc) Stderr() string { return p.stderr.String() }

// Shutdown closes stdin, lets the server exit on its own for up to Timeout (dropping its
// unread output), then kills it if it still runs.
func (p *Proc) Shutdown() {
	p.CloseStdin()
	deadline := time.After(Timeout)
	for {
		select {
		case _, ok := <-p.lines:
			if ok {
				continue
			}
			select {
			case <-p.done:
			case <-deadline:
				p.Kill()
			}
			return
		case <-deadline:
			p.Kill()
			return
		}
	}
}

// Kill stops the process if it still runs and reaps it; it is idempotent.
func (p *Proc) Kill() {
	select {
	case <-p.done:
		return
	default:
	}
	_ = p.cmd.Process.Kill()
	for range p.lines { // drain, so read can reach Wait
	}
	<-p.done
}

// syncBuffer is a bytes.Buffer safe for the writer goroutine of exec and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
