package readtools_test

// n4_read_alarms, retro 2026-11-16 M2: counts per state over every matching record, relative
// since/until and the source_glob filter, all through the in-memory station.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// fixedNow pins the wall clock relative times count back from, restored after the test.
func fixedNow(t *testing.T, at string) {
	t.Helper()
	now, err := time.Parse(time.RFC3339, at)
	if err != nil {
		t.Fatal(err)
	}
	saved := bql.Now
	bql.Now = func() time.Time { return now }
	t.Cleanup(func() { bql.Now = saved })
}

func TestReadAlarmsCountsEveryMatchingRecordPerState(t *testing.T) {
	out := callOK(t, alarmServer(t, 60), "n4_read_alarms", "limit", pyjson.IntLiteral("5"))
	counts, _ := field(out, "state_counts").(*pyjson.Object)
	if counts == nil {
		t.Fatalf("no state_counts: %s", pyjson.Repr(out))
	}
	want := "{'source_state': {'normal': 0, 'offnormal': 60, 'fault': 0, 'alert': 0}, " +
		"'ack_state': {'acked': 20, 'unacked': 40, 'ackPending': 0}, 'records_counted': 60, 'complete': True}"
	if got := pyjson.Repr(counts); got != want {
		t.Errorf("state_counts %s, want %s", got, want)
	}
	if field(out, "row_count") != 5 || field(out, "truncated") != true {
		t.Errorf("the rows stay capped by limit: %s", pyjson.Repr(out))
	}
}

func TestReadAlarmsAcceptsRelativeTimesAndASourceGlob(t *testing.T) {
	fixedNow(t, "2026-11-16T12:00:00Z")
	s := alarmServer(t, 1)
	callOK(t, s, "n4_read_alarms", "source_glob", "*slot:/ImplG/N/*", "since", "24h", "until", "30m")
	where := "source like '*slot:/ImplG/N/*' and (ackState != 'acked' or sourceState != 'normal') and " +
		"timestamp.millis >= 1794744000000 and timestamp.millis <= 1794828600000"
	if got := sent(s)[0]; got != alarmSelect+" where "+where+alarmOrder {
		t.Errorf("ORD sent %q\nwant %q", got, alarmSelect+" where "+where+alarmOrder)
	}
	for _, c := range []struct {
		kv   []any
		want string
	}{
		{[]any{"source_glob", "*slot:/x'y*"}, "source_glob must be a like pattern"},
		{[]any{"source_glob", "local:|station:|slot:/A*"}, "source_glob must be a like pattern"},
		{[]any{"source_glob", "slot:/A B*"}, "source_glob must be a like pattern"},
		{[]any{"since", "24x"}, "since must be a BAbsTime"},
		{[]any{"since", "0h"}, "since must be a BAbsTime"},
		{[]any{"since", "1h", "until", "2h"}, "since must not be later than until"},
	} {
		s := alarmServer(t, 1)
		text := callFail(t, s, "n4_read_alarms", c.kv...)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(text, c.want) {
			t.Errorf("%v: %s (%d requests)", c.kv, text, len(s.Station.OrdRequests))
		}
	}
}

func TestReadAlarmsSaysWhenTheStateCountsAreUnavailable(t *testing.T) {
	s := alarmServer(t, 2)
	s.Station.OrdRoutes[1].Code = 400 // the sourceState.encodeToString route
	out := callOK(t, s, "n4_read_alarms")
	note, _ := field(out, "state_counts_note").(string)
	if v, ok := out.Get("state_counts"); !ok || v != nil || !strings.Contains(note, "HTTP 400") || field(out, "total") != 2 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}
