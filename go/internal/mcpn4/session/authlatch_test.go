package session_test

// Go twins of the test_auth_latch tests (audit 2026-10-03 F4) that swap the clock, the
// environment or the client in-process (go_server.go_twin("session", "injection")). The write
// twin, TestAWriteDryRunIsRefusedWhilePaused, comes with the write tools (slice S10).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

func TestARejectedLoginPausesTheNextConnectWithTheSameCredentials(t *testing.T) {
	s := server()
	wrongPassword(s, "wrong-pw-1234")
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	before := s.Station.Requests
	text := mustErr(t, s, "n4_connect", "station", "FakeStation")
	contains(t, text, "authentication failure", "HTTP 401", "30 s")
	if strings.Contains(text, "wrong-pw-1234") || s.Station.Requests != before {
		t.Errorf("leaked or reached the station: %s", text)
	}
}

func TestAfterTheCooldownOneMoreAttemptIsAllowed(t *testing.T) {
	s := server()
	wrongPassword(s, "wrong-pw-1234")
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	s.Now += 29
	contains(t, mustErr(t, s, "n4_connect", "station", "FakeStation"), "retry in 1 s")
	s.Now++
	wrongPassword(s, readtoolstest.Password)
	s.Connect(t)
	if _, _, desc := s.Call(t, "n4_describe_session"); has(desc, "auth_paused") {
		t.Error("still paused after the cooldown")
	}
}

func TestOtherCredentialsMayReconnectDuringTheCooldown(t *testing.T) {
	s := server()
	wrongPassword(s, "wrong-pw-1234")
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	wrongPassword(s, readtoolstest.Password)
	s.Connect(t)
	if block := s.Ctx.AuthBlock(); block != "" { // a good login clears the latch
		t.Errorf("still latched: %s", block)
	}
}

func TestASecondRejectedCredentialIsLatchedByTheFallback(t *testing.T) {
	st := readtoolstest.NewStation()
	st.NoHook = true
	s := readtoolstest.NewServer(st, session.Config{})
	wrongPassword(s, "wrong-pw-1234")
	mustErr(t, s, "n4_connect", "station", "FakeStation")
	wrongPassword(s, "another-wrong-pw")
	contains(t, mustErr(t, s, "n4_connect", "station", "FakeStation"), "HTTP 401")
	before := st.Requests
	contains(t, mustErr(t, s, "n4_connect", "station", "FakeStation"), "authentication failure")
	if st.Requests != before {
		t.Error("the latched credentials reached the station")
	}
}

// failingReads serves n4_navigate, n4_read_slots and n4_bql_query with a handler that counts its
// calls and answers an AuthError while rejected is set (the station rotated the password).
func failingReads(s *readtoolstest.Server, calls *int, rejected *bool) {
	h := func(*pyjson.Object) (*pyjson.Object, error) {
		*calls++
		if *rejected {
			return nil, &box.Error{Kind: box.KindAuth, Msg: box.AuthMessage(401), Channel: box.Channel, Key: "callssc"}
		}
		return pyjson.NewObject(), nil
	}
	for _, name := range []string{"n4_navigate", "n4_read_slots", "n4_bql_query"} {
		s.Handlers[name] = h
	}
}

func TestARejectedBoxCallPausesEveryLaterStationRead(t *testing.T) {
	s := server()
	s.Connect(t)
	calls, rejected := 0, true
	failingReads(s, &calls, &rejected)
	contains(t, mustErr(t, s, "n4_navigate", "ord", "station:|slot:/"), "HTTP 401")
	for _, call := range [][]any{{"n4_navigate", "ord", "station:|slot:/"}, {"n4_read_slots", "ord", "station:|slot:/Folder"},
		{"n4_bql_query", "query", "select name from control:NumericWritable"}} {
		contains(t, mustErr(t, s, call[0].(string), call[1:]...), "authentication failure", "retry in 30 s")
	}
	if calls != 1 {
		t.Errorf("%d station calls, want 1", calls)
	}
	_, _, desc := s.Call(t, "n4_describe_session")
	paused, _ := desc.Get("auth_paused")
	if got := pyjson.Repr(paused); !strings.Contains(got, "'retry_in_s': 30") || !strings.Contains(got, "HTTP 401") {
		t.Errorf("auth_paused %s", got)
	}
}

func TestARejectedOrdGetPausesToo(t *testing.T) {
	s := server()
	s.Connect(t)
	calls, rejected := 0, true
	failingReads(s, &calls, &rejected)
	contains(t, mustErr(t, s, "n4_bql_query", "query", "select name from control:NumericWritable"), "HTTP 401")
	contains(t, mustErr(t, s, "n4_navigate"), "authentication failure")
}

func TestTheReadsResumeAfterTheCooldown(t *testing.T) {
	s := server()
	s.Connect(t)
	calls, rejected := 0, true
	failingReads(s, &calls, &rejected)
	mustErr(t, s, "n4_navigate", "ord", "station:|slot:/")
	rejected = false
	s.Now += 30
	if text, isErr, _ := s.Call(t, "n4_navigate", "ord", "station:|slot:/"); isErr || calls != 2 {
		t.Errorf("not resumed: %s (calls %d)", text, calls)
	}
}

func TestAnAboutRejectionDuringConnectPausesLaterStationCalls(t *testing.T) {
	st := readtoolstest.NewStation()
	st.AboutErr = &box.Error{Kind: box.KindAuth, Msg: box.AuthMessage(401), Channel: "obix", Key: "about"}
	s := readtoolstest.NewServer(st, session.Config{})
	out := s.Connect(t) // the version probe never fails the connect
	verr, _ := out.Get("version_error")
	if !strings.Contains(verr.(string), "HTTP 401") || !has(out, "auth_paused") {
		t.Errorf("connect %s", pyjson.Repr(out))
	}
	contains(t, mustErr(t, s, "n4_navigate"), "authentication failure")
}
