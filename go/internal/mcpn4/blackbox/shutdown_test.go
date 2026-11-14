package blackbox

// The server process deletes the station's BOX session however it is stopped (port of
// test_server.TestShutdown; docs/go-port-plan.md section 22).

import (
	"os"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// liveProc is a fake station and a server process for it, nothing sent yet.
func liveProc(t *testing.T) (*boxstation.Station, *Proc) {
	st := boxstation.New(User, Password, StationName).Start(t)
	return st, Spawn(t, Options{AllowHTTP: true, Stations: map[string]string{StationName: st.URL},
		Env: map[string]string{"MCP_N4_USER": User, "MCP_N4_PASSWORD": Password}})
}

func sendConnect(t *testing.T, p *Proc) {
	t.Helper()
	if err := p.Send(RPC("tools/call", Args{"name": "n4_connect", "arguments": Args{"station": StationName}}, 1)); err != nil {
		t.Fatal(err)
	}
}

// assertStoppedClean checks a signal with stdin still open ends the server with exit 0, no
// output after it, and no station session left.
func assertStoppedClean(t *testing.T, st *boxstation.Station, p *Proc, sig os.Signal) {
	t.Helper()
	if err := p.Signal(sig); err != nil {
		t.Fatal(err)
	}
	code, rest, stderr := p.Wait(t)
	if code != 0 || rest != nil || stderr != "" || len(st.Sessions()) != 0 {
		t.Errorf("exit %d, stdout %q, stderr %q, sessions %v", code, rest, stderr, st.Sessions())
	}
}

func assertSignalClosesTheSession(t *testing.T, sig os.Signal) {
	st, p := liveProc(t)
	sendConnect(t, p)
	if line, _ := p.ReadLine(); !strings.Contains(line, `"station_name"`) {
		t.Fatalf("connect reply %q", line)
	}
	if got := st.Sessions(); !reflect.DeepEqual(got, []string{"sess-1"}) {
		t.Fatalf("sessions %v", got)
	}
	assertStoppedClean(t, st, p, sig)
}

func TestSigintDeletesTheStationSessionAndExitsZero(t *testing.T) {
	assertSignalClosesTheSession(t, syscall.SIGINT)
}

func TestSigtermDeletesTheStationSessionAndExitsZero(t *testing.T) {
	assertSignalClosesTheSession(t, syscall.SIGTERM)
}

func TestSigintAfterMakeStillDeletesTheStationSession(t *testing.T) {
	st, p := liveProc(t)
	reached, release := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release) })
	st.Hook = func(frame map[string]any) (int, []byte) {
		if msgs, _ := frame["m"].([]any); len(msgs) > 0 && msgs[0].(map[string]any)["k"] == "makessc" {
			close(reached)
			select {
			case <-release:
			case <-time.After(Timeout):
			}
		}
		return 0, nil
	}
	sendConnect(t, p)
	select {
	case <-reached:
	case <-time.After(Timeout):
		t.Fatal("makessc never reached the station")
	}
	if got := st.Sessions(); !reflect.DeepEqual(got, []string{"sess-1"}) {
		t.Fatalf("sessions %v", got)
	}
	assertStoppedClean(t, st, p, syscall.SIGINT)
}

func TestAClientGoneMidReplyStillDeletesTheStationSession(t *testing.T) {
	st, p := liveProc(t)
	sendConnect(t, p)
	p.ReadLine()
	p.CloseStdout()
	_ = p.Send(RPC("ping", nil, 1)) // its reply cannot be written: EPIPE, not a SIGPIPE death
	code, _, _ := p.Wait(t)
	if code <= 0 || len(st.Sessions()) != 0 { // an exit status (Go 1), never a signal (-1)
		t.Errorf("exit %d, sessions %v", code, st.Sessions())
	}
}
