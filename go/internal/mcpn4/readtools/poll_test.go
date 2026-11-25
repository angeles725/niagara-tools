package readtools_test

// n4_poll (#891) through the in-memory station: a fake clock and sleep, so no test waits.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const pollStationTime = "2026-11-25T12:00:10.000-06:00"

// pollServer answers the select with bodies[i] at the i-th sample; each sleep advances the fake
// clocks, and the local wall clock is 10 s behind the station.
func pollServer(t *testing.T, bodies ...string) (*readtoolstest.Server, *[]float64) {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.Trees = map[string]*pyjson.Object{"station:|slot:/Services/PlatformServices/SystemService": nodes(
		[]any{"", "h", "s1"}, []any{"systemTime", "t", "baja:AbsTime", "v", pollStationTime})}
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "from control:ControlPoint", Body: bodies[0]}}
	wall, _ := time.Parse(time.RFC3339, "2026-11-25T11:43:20-06:00") // + Now (1000 s) = 12:00:00
	slept, i := &[]float64{}, 0
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, NewClient: s.Station.Factory, ServerVersion: "test",
		Now: func() time.Time { return wall.Add(time.Duration(s.Now * float64(time.Second))) },
		Sleep: func(sec float64) {
			*slept = append(*slept, sec)
			s.Now += sec
			if i++; i < len(bodies) {
				s.Station.OrdRoutes[0].Body = bodies[i]
			}
		}})
	return s, slept
}

func TestPollReadsEachSampleOnScheduleAndSummarizesTheCells(t *testing.T) {
	s, slept := pollServer(t, "Slot Path,Out\nslot:/A,21.0 °C\n", "Slot Path,Out\nslot:/A,\"21,5 °C\"\n",
		"Slot Path,Out\nslot:/A,20.5 °C\n")
	out := callOK(t, s, "n4_poll", "query", "select slotPath, out from control:ControlPoint", "samples", pyjson.IntLiteral("3"),
		"interval_s", pyjson.IntLiteral("2"))
	samples, _ := out.Get("samples")
	if len(samples.([]any)) != 3 || len(*slept) != 2 || (*slept)[0] != 2 || pyjson.Str(get(out, "ended")) != "complete" {
		t.Fatalf("slept %v: %s", *slept, pyjson.Repr(out))
	}
	series, _ := out.Get("series")
	cell := series.([]any)[0].(*pyjson.Object)
	want := `{"row": "slot:/A", "column": "Out", "first": "21.0 °C", "last": "20.5 °C", "changes": 2, "min": 20.5, "max": 21.5}`
	if pyjson.Repr(cell) != pyjson.Repr(mustObj(t, want)) {
		t.Errorf("cell %s", pyjson.Repr(cell))
	}
	clock, _ := out.Get("clock")
	if get(clock.(*pyjson.Object), "station_time") != pollStationTime || get(clock.(*pyjson.Object), "offset_s") != 10.0 {
		t.Errorf("clock %s", pyjson.Repr(clock))
	}
}

func TestPollRefusesASpanPastTheClientLimit(t *testing.T) {
	s, _ := pollServer(t, "Slot Path,Out\n")
	text := callFail(t, s, "n4_poll", "query", "select slotPath, out from control:ControlPoint", "samples", pyjson.IntLiteral("11"),
		"interval_s", pyjson.IntLiteral("5"))
	if !strings.Contains(text, "spans 50 s; at most 40 s") {
		t.Error(text)
	}
}

// An authentication failure ends the poll: another read would be one more failed login toward
// the station's lockout.
func TestPollStopsAtAnAuthenticationFailure(t *testing.T) {
	s, _ := pollServer(t, "Slot Path,Out\nslot:/A,1\n")
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "from control:NumericPoint", Code: 401}}
	out := callOK(t, s, "n4_poll", "query", "select slotPath, out from control:NumericPoint", "samples", pyjson.IntLiteral("3"))
	samples, _ := out.Get("samples")
	if pyjson.Str(get(out, "ended")) != "auth_failed" || len(samples.([]any)) != 1 || len(s.Station.OrdRequests) != 1 {
		t.Fatal(pyjson.Repr(out))
	}
}

func get(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }

func mustObj(t *testing.T, s string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}
