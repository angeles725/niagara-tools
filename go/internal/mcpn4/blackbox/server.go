package blackbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

// ErrClosed is the error of an exchange with a closed Server.
var ErrClosed = errors.New("blackbox: the server is closed")

// StartupError is a server that refused its options and exited before serving.
type StartupError struct {
	Code   int
	Stderr string
}

func (e *StartupError) Error() string {
	return fmt.Sprintf("the server exited %d at start: %s", e.Code, e.Stderr)
}

// Server is a long-lived mcp-n4 process spoken to one JSON line per request. Each exchange
// writes the line, then a ping sentinel, and reads up to the sentinel's reply: zero lines
// before it mean "no reply" (a notification), one is the reply, more is a protocol error.
type Server struct {
	t      testing.TB
	p      *Proc
	sync   int
	closed bool
}

// Start is StartErr that fails the test when the server does not start.
func Start(t testing.TB, o Options) *Server {
	t.Helper()
	s, err := StartErr(t, o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// StartErr starts the server and waits until it answers; a server that exits first is a
// *StartupError carrying its exit code and stderr. The test's cleanup closes it.
func StartErr(t testing.TB, o Options) (*Server, error) {
	t.Helper()
	s := &Server{t: t, p: Spawn(t, o)}
	t.Cleanup(s.Close)
	if _, _, err := s.Exchange(""); err != nil {
		s.Close()
		code, _, stderr := s.p.Wait(t)
		return nil, &StartupError{Code: code, Stderr: stderr}
	}
	return s, nil
}

// Proc is the process under the server, for signals and exit status.
func (s *Server) Proc() *Proc { return s.p }

// Exchange sends line ("" sends only the sentinel) and returns the server's reply line; ok is
// false when there was no reply.
func (s *Server) Exchange(line string) (reply string, ok bool, err error) {
	if s.closed {
		return "", false, ErrClosed
	}
	s.sync++
	id := fmt.Sprintf("__blackbox_sync_%d", s.sync)
	sentinel := fmt.Sprintf(`{"jsonrpc":"2.0","id":%q,"method":"ping"}`, id)
	payload := sentinel
	if line != "" {
		payload = line + "\n" + sentinel
	}
	if err := s.p.Send(payload); err != nil {
		return "", false, fmt.Errorf("server stdin: %w; stderr: %s", err, s.p.Stderr())
	}
	var replies []string
	for {
		out, more := s.p.ReadLine()
		if !more {
			return "", false, fmt.Errorf("no reply (server exited or %s passed); stderr: %s", Timeout, s.p.Stderr())
		}
		var obj map[string]any
		if json.Unmarshal([]byte(out), &obj) == nil && obj["id"] == id {
			break
		}
		replies = append(replies, out)
	}
	switch len(replies) {
	case 0:
		return "", false, nil
	case 1:
		return replies[0], true, nil
	}
	return "", false, fmt.Errorf("one request produced %d reply lines: %q", len(replies), replies)
}

// Line is Exchange that fails the test on a protocol error; "" means no reply.
func (s *Server) Line(line string) string {
	s.t.Helper()
	reply, _, err := s.Exchange(line)
	if err != nil {
		s.t.Fatal(err)
	}
	return reply
}

// Close ends stdin and waits for the exit (the server deletes its station session on EOF); it
// is idempotent.
func (s *Server) Close() {
	if s.closed {
		return
	}
	s.closed = true
	s.p.Shutdown()
}
