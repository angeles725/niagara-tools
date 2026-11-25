package readtools_test

// n4_subscribe_point (#875) through the in-memory station: a fake clock and sleep, so no test waits.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const subPoint = "station:|slot:/Drivers/Net/Dev/points/Temp"

// proxyPoint is the load of a proxy point whose out and proxy ext carry status (hex bits).
func proxyPoint(value any, status string) *pyjson.Object {
	return nodes([]any{"", "h", "p1", "t", "control:NumericPoint"},
		[]any{"out", "t", "baja:StatusNumeric"}, []any{"out/value", "v", value}, []any{"out/status", "v", status},
		[]any{"proxyExt", "t", "modbusAsync:ModbusClientNumericProxyExt"}, []any{"proxyExt/status", "v", status})
}

// subServer answers the point's load with loads[i] after the i-th sleep (the last one repeats);
// each sleep advances the fake clock.
func subServer(t *testing.T, loads ...*pyjson.Object) (*readtoolstest.Server, *[]float64) {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.Trees = map[string]*pyjson.Object{subPoint: loads[0]}
	slept, i := &[]float64{}, 0
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, NewClient: s.Station.Factory, ServerVersion: "test",
		Sleep: func(sec float64) {
			*slept = append(*slept, sec)
			s.Now += sec
			if i++; i < len(loads) {
				s.Station.Trees[subPoint] = loads[i]
			}
		}})
	return s, slept
}

func TestSubscribePointEndsWhenTheStaleFlagClears(t *testing.T) {
	s, slept := subServer(t, proxyPoint(0.0, "10"), proxyPoint(0.0, "10"), proxyPoint(21.5, "0"))
	out := callOK(t, s, "n4_subscribe_point", "ord", subPoint, "seconds", pyjson.IntLiteral("10"))
	if get(out, "ended") != "stale_cleared" || len(*slept) != 2 || get(out, "waited_s") != 4.0 {
		t.Fatalf("slept %v: %s", *slept, pyjson.Repr(out))
	}
	after := get(out, "after").(*pyjson.Object)
	if get(after, "value") != 21.5 || pyjson.Repr(get(after, "status_flags")) != "[]" {
		t.Errorf("after %s", pyjson.Repr(after))
	}
	before := get(out, "before").(*pyjson.Object)
	if pyjson.Repr(get(before, "status_flags")) != `['stale']` || get(before, "proxy_ext") != "modbusAsync:ModbusClientNumericProxyExt" {
		t.Errorf("before %s", pyjson.Repr(before))
	}
	want := []string{"sub " + subPoint, "unsub " + subPoint}
	if got := s.Station.SSCCalls(); strings.Join(got, "|") != strings.Join(want, "|") || get(out, "unsubscribed") != true {
		t.Errorf("calls %v: %s", got, pyjson.Repr(out))
	}
}

func TestSubscribePointStillStaleAtTheCapSaysWhy(t *testing.T) {
	s, slept := subServer(t, proxyPoint(0.0, "10"))
	out := callOK(t, s, "n4_subscribe_point", "ord", subPoint, "seconds", pyjson.IntLiteral("5"))
	if get(out, "ended") != "timeout" || len(*slept) != 3 || (*slept)[2] != 1 || get(out, "waited_s") != 5.0 {
		t.Fatalf("slept %v: %s", *slept, pyjson.Repr(out))
	}
	if !strings.Contains(pyjson.Repr(get(out, "notes")), "still stale after 5 s") {
		t.Errorf("notes %s", pyjson.Repr(out))
	}
}

func TestSubscribePointOnADownPointNamesTheDevice(t *testing.T) {
	s, slept := subServer(t, proxyPoint(0.0, "14")) // down + stale
	out := callOK(t, s, "n4_subscribe_point", "ord", subPoint, "seconds", pyjson.IntLiteral("10"))
	if get(out, "ended") != "not_polled" || len(*slept) != 1 {
		t.Fatalf("slept %v: %s", *slept, pyjson.Repr(out))
	}
	if !strings.Contains(pyjson.Repr(get(out, "notes")), "the point is {down, stale}") || len(s.Station.SSCCalls()) != 2 {
		t.Errorf("%v: %s", s.Station.SSCCalls(), pyjson.Repr(out))
	}
}

func TestSubscribePointUnsubscribesWhenAReadFails(t *testing.T) {
	s, _ := subServer(t, proxyPoint(0.0, "10"))
	// the point vanishes at the first check: the read fails, the unsub is still sent
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, NewClient: s.Station.Factory, ServerVersion: "test",
		Sleep: func(sec float64) { s.Now += sec; delete(s.Station.Trees, subPoint) }})
	out := callOK(t, s, "n4_subscribe_point", "ord", subPoint, "seconds", pyjson.IntLiteral("4"))
	if get(out, "ended") != "read_failed" || get(out, "error") == nil || get(out, "unsubscribed") != true {
		t.Fatalf("%s", pyjson.Repr(out))
	}
	if got := s.Station.SSCCalls(); len(got) != 2 || got[1] != "unsub "+subPoint {
		t.Errorf("calls %v", got)
	}
}

func TestSubscribePointRefusesItsBoundsAndANonPoint(t *testing.T) {
	s, _ := subServer(t, proxyPoint(0.0, "10"))
	// the shared validator refuses seconds outside the schema's 1 to 40 before the tool runs
	for secs, want := range map[string]string{"0": "must be >= 1", "41": "must be <= 40"} {
		if msg := rpcError(t, s, "n4_subscribe_point", "ord", subPoint, "seconds", pyjson.IntLiteral(secs)); !strings.Contains(msg, want) {
			t.Errorf("seconds %s: %s", secs, msg)
		}
	}
	s.Station.Trees["station:|slot:/Folder"] = nodes([]any{"", "h", "f1", "t", "baja:Folder"})
	if msg := callFail(t, s, "n4_subscribe_point", "ord", "station:|slot:/Folder"); !strings.Contains(msg, "has no out slot") {
		t.Errorf("folder: %s", msg)
	}
	if n := len(s.Station.SSCCalls()); n != 0 {
		t.Errorf("a refused call subscribed: %d calls", n)
	}
}

func TestSubscribePointOnAPointThatWasNeverStaleSaysSo(t *testing.T) {
	s, slept := subServer(t, proxyPoint(20.0, "0"), proxyPoint(20.5, "0"))
	out := callOK(t, s, "n4_subscribe_point", "ord", subPoint)
	if get(out, "ended") != "not_stale" || len(*slept) != 1 || get(get(out, "after").(*pyjson.Object), "value") != 20.5 {
		t.Fatalf("slept %v: %s", *slept, pyjson.Repr(out))
	}
}
