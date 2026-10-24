package session_test

// Go twins of the test_server.TestConnect tests that reach srv.ctx or client_factory in-process
// (go_server.go_twin("session", ...)).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func server() *readtoolstest.Server {
	return readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
}

func wrongPassword(s *readtoolstest.Server, pw string) { s.Env["MCP_N4_PASSWORD"] = pw }

func mustErr(t *testing.T, s *readtoolstest.Server, name string, kv ...any) string {
	t.Helper()
	text, isErr, _ := s.Call(t, name, kv...)
	if !isErr {
		t.Fatalf("%s succeeded", name)
	}
	return text
}

func has(o *pyjson.Object, key string) bool {
	_, ok := o.Get(key)
	return ok
}

func contains(t *testing.T, text string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(text, n) {
			t.Errorf("%q lacks %q", text, n)
		}
	}
}

func TestMissingEnvVarsNameTheVariablesNotValues(t *testing.T) {
	s := server()
	s.Env = map[string]string{}
	contains(t, mustErr(t, s, "n4_connect", "station", "FakeStation"), "MCP_N4_USER", "MCP_N4_PASSWORD")
	if s.Station.Requests != 0 {
		t.Errorf("requests %d", s.Station.Requests)
	}
}

func TestWrongPasswordIsAnErrorWithoutLeakingTheSecret(t *testing.T) {
	s := server()
	wrongPassword(s, "wrong-pw-1234")
	if text := mustErr(t, s, "n4_connect", "station", "FakeStation"); strings.Contains(text, "wrong-pw-1234") {
		t.Errorf("leaked: %s", text)
	}
	if s.Station.Requests != 1 { // no login retry (B1179)
		t.Errorf("requests %d", s.Station.Requests)
	}
	contains(t, mustErr(t, s, "n4_navigate"), "not connected")
}

func TestExpectedStationDefaultsToTheStationName(t *testing.T) {
	s := server()
	out := s.Connect(t)
	if name, _ := out.Get("station_name"); name != "FakeStation" || !s.Ctx.Session.IdentityVerified {
		t.Errorf("connect %s, session %+v", pyjson.Repr(out), s.Ctx.Session)
	}
}

func TestDefaultExpectedNameMismatchIsRefused(t *testing.T) {
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{Stations: map[string]string{"Other": "http://h/"}})
	contains(t, mustErr(t, s, "n4_connect", "station", "Other"), "'Other'", "'FakeStation'")
	if s.Ctx.Session != nil || len(s.Station.Sessions) != 0 {
		t.Error("the mismatched session was kept")
	}
}

func TestExpectedStationMismatchClosesSessionAndNamesBoth(t *testing.T) {
	s := server()
	text := mustErr(t, s, "n4_connect", "station", "FakeStation", "expected_station", "OtherStation")
	if want := "station identity mismatch: expected 'OtherStation' but connected to 'FakeStation'"; text != want {
		t.Errorf("got %q", text)
	}
	if s.Ctx.Session != nil || len(s.Station.Sessions) != 0 {
		t.Error("the mismatched session was kept")
	}
	contains(t, mustErr(t, s, "n4_navigate"), "not connected")
}

func TestFailedConnectOnUnknownStationClosesTheOldSession(t *testing.T) {
	s := server()
	s.Connect(t)
	contains(t, mustErr(t, s, "n4_connect", "station", "ghost"), "'ghost'", "FakeStation", "--station")
	if s.Ctx.Session != nil || len(s.Station.Sessions) != 0 {
		t.Error("the old session survived")
	}
}

func TestFailedConnectOnMissingCredentialsClosesTheOldSession(t *testing.T) {
	s := server()
	s.Connect(t)
	s.Env = map[string]string{}
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	if s.Ctx.Session != nil || len(s.Station.Sessions) != 0 {
		t.Error("the old session survived")
	}
	contains(t, mustErr(t, s, "n4_navigate"), "not connected")
}

func TestReconnectClosesThePreviousSession(t *testing.T) {
	s := server()
	s.Connect(t)
	first := s.Ctx.Session
	s.Connect(t)
	if s.Ctx.Session == first || len(s.Station.Sessions) != 1 || s.Station.Sessions[1] {
		t.Errorf("sessions %v", s.Station.Sessions)
	}
}

func TestFailedReconnectLeavesNoSession(t *testing.T) {
	s := server()
	s.Connect(t)
	wrongPassword(s, "wrong-pw-1234")
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	if s.Ctx.Session != nil {
		t.Error("a session survived the failed reconnect")
	}
}

func TestTlsIsVerifiedUnlessTheOperatorMarksTheStationInsecure(t *testing.T) {
	for _, tc := range []struct {
		insecure []string
		want     bool
	}{{nil, false}, {[]string{"FakeStation"}, true}} {
		s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{InsecureTLS: tc.insecure,
			HTTPTimeout: 90, LoadWait: 12.5})
		s.Connect(t)
		cfg := s.Station.Configs[0]
		if cfg.InsecureTLS != tc.want || cfg.Timeout != 90 || cfg.LoadWait != 12.5 || !cfg.AllowHTTP ||
			cfg.Username != "admin" || cfg.BaseURL != "http://fake.invalid/" {
			t.Errorf("insecure %v: client config %+v", tc.insecure, cfg)
		}
	}
}

func TestInsecureTlsForAnUnconfiguredStationIsAStartError(t *testing.T) {
	cfg := cli.Defaults()
	cfg.AllowHTTP, cfg.Stations, cfg.InsecureTLS = true, []string{"FakeStation=http://h"}, []string{"ghost"}
	if err := cli.Check(cfg); err == nil || err.Error() != "--insecure-tls names unconfigured station(s): ghost" {
		t.Errorf("Check = %v", err)
	}
}
