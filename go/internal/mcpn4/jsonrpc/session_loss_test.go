package jsonrpc

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// lost is the station's reply when it dropped this server's BOX session (seen four times in one
// afternoon on a 4.14 station behind a tunnel).
var lost = &box.Error{Kind: box.KindBox, Msg: "Could not find Server Session", Station: true}

const callConnect = `{"id": 1, "method": "tools/call", "params": {"name": "n4_connect", "arguments": {"station": "S1"}}}`

// lossServer serves n4_connect (counting calls, failing with connectErr) and h as n4_navigate and
// n4_discover_devices.
func lossServer(h Handler, connectErr error) (*Server, *[]string) {
	s := navigateServer(&fakeContext{connected: true}, h)
	s.EnableWrites()
	s.Handlers["n4_discover_devices"] = h
	var connects []string
	s.Handlers["n4_connect"] = func(args *pyjson.Object) (*pyjson.Object, error) {
		st, _ := args.Get("station")
		connects = append(connects, pyjson.Str(st))
		if connectErr != nil {
			return nil, connectErr
		}
		return obj("station_name", "S1"), nil
	}
	return s, &connects
}

// failsOnce answers lost on its first call and ok after.
func failsOnce(calls *int) Handler {
	return func(*pyjson.Object) (*pyjson.Object, error) {
		*calls++
		if *calls == 1 {
			return nil, lost
		}
		return obj("ok", true), nil
	}
}

// A read that finds its session gone reconnects with the last n4_connect arguments and runs once more.
func TestALostSessionReconnectsAndRetriesARead(t *testing.T) {
	calls := 0
	s, connects := lossServer(failsOnce(&calls), nil)
	handle(t, s, callConnect)
	got := handle(t, s, callNavigate)
	if !strings.Contains(got, `"structuredContent": {"ok": true}`) || calls != 2 || len(*connects) != 2 || (*connects)[1] != "S1" {
		t.Errorf("got %s; calls %d, connects %v", got, calls, *connects)
	}
}

// A dry run sends nothing to the station either: it is retried the same way.
func TestALostSessionRetriesADryRun(t *testing.T) {
	calls := 0
	s, connects := lossServer(failsOnce(&calls), nil)
	handle(t, s, callConnect)
	got := handle(t, s, `{"id": 2, "method": "tools/call", "params": {"name": "n4_discover_devices", "arguments": {"network_ord": "station:|slot:/Drivers/BacnetNetwork"}}}`)
	if !strings.Contains(got, `"structuredContent": {"ok": true}`) || calls != 2 || len(*connects) != 2 {
		t.Errorf("got %s; calls %d, connects %v", got, calls, *connects)
	}
}

// An execution is never retried: its token belonged to the lost session. The server reconnects
// and says to redo the dry run.
func TestALostSessionDuringAnExecutionAsksForANewDryRun(t *testing.T) {
	calls := 0
	s, connects := lossServer(failsOnce(&calls), nil)
	handle(t, s, callConnect)
	got := handle(t, s, `{"id": 2, "method": "tools/call", "params": {"name": "n4_discover_devices", "arguments": {"network_ord": "station:|slot:/Drivers/BacnetNetwork", "dry_run": false, "confirmation_token": "t"}}}`)
	for _, part := range []string{"Could not find Server Session", "reconnected", "may have applied", "n4_list_batches", "redo the dry run", `"isError": true`} {
		if !strings.Contains(got, part) {
			t.Errorf("missing %q: %s", part, got)
		}
	}
	if calls != 1 || len(*connects) != 2 {
		t.Errorf("calls %d, connects %v; want 1 call and one reconnect", calls, *connects)
	}
}

// A failed reconnect is reported with the original loss; without an earlier n4_connect, and for
// any other station error, the error is passed through unchanged.
func TestALostSessionEdgeCases(t *testing.T) {
	calls := 0
	s, connects := lossServer(failsOnce(&calls), errors.New("x"))
	s.Handlers["n4_connect"] = func(*pyjson.Object) (*pyjson.Object, error) {
		*connects = append(*connects, "S1")
		if len(*connects) == 1 {
			return obj("station_name", "S1"), nil
		}
		return nil, &box.Error{Kind: box.KindBox, Msg: "connection refused"}
	}
	handle(t, s, callConnect)
	got := handle(t, s, callNavigate)
	if !strings.Contains(got, "Could not find Server Session") || !strings.Contains(got, "reconnect") ||
		!strings.Contains(got, "connection refused") || calls != 1 {
		t.Errorf("failed reconnect: %s (calls %d)", got, calls)
	}

	calls = 0
	s, connects = lossServer(failsOnce(&calls), nil)
	if got := handle(t, s, callNavigate); !strings.Contains(got, `"text": "Could not find Server Session"`) || len(*connects) != 0 {
		t.Errorf("no earlier connect: %s, connects %v", got, *connects)
	}

	other := 0
	s, connects = lossServer(func(*pyjson.Object) (*pyjson.Object, error) {
		other++
		return nil, &box.Error{Kind: box.KindBox, Msg: "no such component", Station: true}
	}, nil)
	handle(t, s, callConnect)
	if got := handle(t, s, callNavigate); !strings.Contains(got, `"text": "no such component"`) || other != 1 || len(*connects) != 1 {
		t.Errorf("other error: %s, calls %d, connects %v", got, other, *connects)
	}
}

// A tool that passes the loss on as its own error text is recognised too.
func TestALostSessionPassedOnAsAToolErrorIsRecognised(t *testing.T) {
	calls := 0
	s, connects := lossServer(func(*pyjson.Object) (*pyjson.Object, error) {
		calls++
		if calls == 1 {
			return nil, &ToolError{Msg: "load failed: " + sessionLostText}
		}
		return obj("ok", true), nil
	}, nil)
	handle(t, s, callConnect)
	if got := handle(t, s, callNavigate); !strings.Contains(got, `{"ok": true}`) || len(*connects) != 2 {
		t.Errorf("got %s, connects %v", got, *connects)
	}
}
