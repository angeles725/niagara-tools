package previewshot

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// stopGrace is how long a server gets to end after SIGTERM before it is killed.
const stopGrace = 2 * time.Second

// freePort asks the OS for a free TCP port on the loopback. The port is released before the server binds it,
// so another process could take it in between; the server then exits and the run reports it (exit 1).
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// findServer: PREVIEW_SERVER, else dashboard-preview on PATH, else the niagara-go install directory.
func findServer(s cli.Stdio) (string, error) {
	if p := s.Env("PREVIEW_SERVER"); p != "" {
		if !isExec(p) {
			return "", errf(exitUsage, "PREVIEW_SERVER=%s is not an executable file", p)
		}
		return p, nil
	}
	if p := onPath(s, "dashboard-preview"); p != "" {
		return p, nil
	}
	if p := filepath.Join(homeDir(s), ".local", "lib", "niagara-go", "bin", "dashboard-preview"); isExec(p) {
		return p, nil
	}
	return "", errf(exitUsage, "no dashboard-preview: set PREVIEW_SERVER, or put it on PATH "+
		"(installed to ~/.local/lib/niagara-go/bin by niagara-research tools/install.sh)")
}

// stateDir holds the PID file of the running server (PREVIEW_SHOT_STATE, else under XDG_STATE_HOME).
func stateDir(s cli.Stdio) string {
	if d := s.Env("PREVIEW_SHOT_STATE"); d != "" {
		return d
	}
	base := s.Env("XDG_STATE_HOME")
	if base == "" {
		base = filepath.Join(homeDir(s), ".local", "state")
	}
	return filepath.Join(base, "niagara-tools", "preview-shot")
}

// syncBuf collects the server's output; it is read while the process may still be writing.
type syncBuf struct {
	mu sync.Mutex
	b  []byte
}

func (w *syncBuf) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.b = append(w.b, p...)
	return len(p), nil
}

func (w *syncBuf) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.b)
}

// server is the one process this run started. It is stopped through the exec.Cmd that started it: Wait has
// not returned until the process is reaped, so its PID cannot belong to anything else, and nothing is ever
// found by a command-line pattern.
type server struct {
	cmd     *exec.Cmd
	done    chan struct{} // closed when the process has ended and been reaped
	log     *syncBuf
	pidFile string
	url     string
}

// startServer starts the preview on a free port, records its PID and waits until the shot URL answers. The
// returned server is non-nil as soon as the process runs, even with an error: the caller must stop it.
func startServer(ctx context.Context, s cli.Stdio, bin string, o options) (*server, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	argv := []string{"--rc", o.serve, "--port", strconv.Itoa(port)}
	if o.prefix != "" {
		argv = append(argv, "--prefix", o.prefix)
	}
	if o.mock != "" {
		argv = append(argv, "--mock", o.mock)
	}
	sv := &server{log: &syncBuf{}, done: make(chan struct{}),
		pidFile: filepath.Join(stateDir(s), "server-"+strconv.Itoa(port)+".pid"),
		url:     "http://127.0.0.1:" + strconv.Itoa(port) + o.path}
	sv.cmd = exec.Command(bin, argv...)
	sv.cmd.Stdout, sv.cmd.Stderr = sv.log, sv.log
	if err := sv.cmd.Start(); err != nil {
		return nil, errf(exitFail, "cannot start %s: %v", bin, err)
	}
	go func() { _ = sv.cmd.Wait(); close(sv.done) }()
	if err := os.MkdirAll(filepath.Dir(sv.pidFile), 0o755); err != nil {
		return sv, err
	}
	if err := os.WriteFile(sv.pidFile, []byte(strconv.Itoa(sv.cmd.Process.Pid)+"\n"), 0o644); err != nil {
		return sv, err
	}
	return sv, sv.waitReady(ctx, wait(s))
}

func wait(s cli.Stdio) time.Duration {
	if n, err := strconv.Atoi(s.Env("PREVIEW_SHOT_WAIT_MS")); err == nil && n > 0 {
		return time.Duration(n) * time.Millisecond
	}
	return 30 * time.Second
}

// waitReady polls the shot URL until it answers with anything (a 404 is an answer), the server dies, the
// limit passes or the run is interrupted.
func (sv *server) waitReady(ctx context.Context, limit time.Duration) error {
	deadline := time.NewTimer(limit)
	defer deadline.Stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	client := &http.Client{Timeout: time.Second}
	for {
		select {
		case <-sv.done:
			return errf(exitFail, "the preview server exited before it answered:\n%s", tail(sv.log.String()))
		case <-ctx.Done():
			return errf(exitFail, "interrupted")
		case <-deadline.C:
			return errf(exitFail, "the preview server did not answer %s within %v:\n%s", sv.url, limit, tail(sv.log.String()))
		case <-tick.C:
			if resp, err := client.Get(sv.url); err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

// stop ends the server this run started (nothing is signalled when it has already ended) and removes the
// PID file: SIGTERM, then SIGKILL after stopGrace.
func (sv *server) stop() {
	defer os.Remove(sv.pidFile)
	select {
	case <-sv.done:
		return
	default:
	}
	_ = sv.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-sv.done:
	case <-time.After(stopGrace):
		_ = sv.cmd.Process.Kill()
		<-sv.done
	}
}
