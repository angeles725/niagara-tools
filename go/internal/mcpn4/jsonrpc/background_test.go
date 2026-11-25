package jsonrpc

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A call that left background work (a write that answered {state: running}, #925) holds the
// next request until that work ends: two calls never use the station session at once.
func TestTheNextRequestWaitsForTheBackgroundWorkOfTheLastOne(t *testing.T) {
	t.Parallel()
	calls, ended := 0, false
	var s *Server
	s = echoServer(func(*pyjson.Object) (*pyjson.Object, error) {
		calls++
		if calls == 1 {
			s.Background.Go(func() { time.Sleep(150 * time.Millisecond); ended = true })
			return obj("state", "running"), nil
		}
		return obj("ended_before", ended), nil
	})
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(callEcho+"\n"+callEcho+"\n"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"structuredContent": {"ended_before": true}`) {
		t.Fatal(out.String())
	}
}

// The loop also waits for background work before it ends (and closes the session).
func TestServeEndsOnlyAfterTheBackgroundWork(t *testing.T) {
	t.Parallel()
	ended := false
	var s *Server
	s = echoServer(func(*pyjson.Object) (*pyjson.Object, error) {
		s.Background.Go(func() { time.Sleep(100 * time.Millisecond); ended = true })
		return obj(), nil
	})
	var out bytes.Buffer
	if err := s.Serve(strings.NewReader(callEcho+"\n"), &out); err != nil || !ended {
		t.Fatalf("%v ended=%v", err, ended)
	}
}
