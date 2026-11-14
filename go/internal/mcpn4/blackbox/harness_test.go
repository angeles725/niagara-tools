package blackbox

// The harness itself (port of mcp-n4-kit/tests/test_go_harness.py, TestGoServerOverStdio): the
// stdio adapter speaks the wire protocol to the real server.

import (
	"errors"
	"strings"
	"testing"
)

func TestTheHarnessReturnsTheReplyObject(t *testing.T) {
	s := Start(t, Options{})
	res := s.Request("initialize", Args{"protocolVersion": "2025-03-26"})
	if name := Dig(res, "result", "serverInfo", "name"); name != "mcp-n4" {
		t.Errorf("initialize = %v", res)
	}
	if got := s.Line(RPC("ping", nil, 7)); got != `{"jsonrpc": "2.0", "id": 7, "result": {}}` {
		t.Errorf("ping = %s", got)
	}
}

func TestTheHarnessReturnsNoReplyForANotificationWithoutBlocking(t *testing.T) {
	s := Start(t, Options{})
	if got := s.Notify("notifications/initialized", nil); got != nil {
		t.Errorf("notification reply = %v", got)
	}
	if got := s.Request("ping", nil)["result"]; !isEmptyObject(got) {
		t.Errorf("ping result = %v", got)
	}
}

func TestTheHarnessPassesRawLinesThrough(t *testing.T) {
	s := Start(t, Options{})
	if code := ErrorCode(s.decode(s.Line("{not json"))); code != -32700 {
		t.Errorf("code = %d", code)
	}
	if got := s.Line("   "); got != "" {
		t.Errorf("blank line reply = %s", got)
	}
}

func TestAStartupRefusalIsAStartupErrorCarryingStderr(t *testing.T) {
	_, err := StartErr(t, Options{Stations: map[string]string{"S": "http://127.0.0.1:9"}})
	var se *StartupError
	if !errors.As(err, &se) || se.Code != 2 || !strings.Contains(se.Stderr, "https") {
		t.Errorf("err = %v", err)
	}
}

func TestCloseIsIdempotentAndAClosedServerRefusesCalls(t *testing.T) {
	s := Start(t, Options{})
	s.Close()
	s.Close()
	if _, _, err := s.Exchange(RPC("ping", nil, 1)); !errors.Is(err, ErrClosed) {
		t.Errorf("err = %v", err)
	}
}

func TestTheHarnessRunsASessionAgainstTheFakeStation(t *testing.T) {
	r := NewRig(t, nil)
	if got := r.Connect(nil)["station_name"]; got != StationName {
		t.Errorf("station_name = %v", got)
	}
}

func isEmptyObject(v any) bool {
	m, ok := v.(map[string]any)
	return ok && len(m) == 0
}
