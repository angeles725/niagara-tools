package readtools

import (
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func row(kv ...string) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i], kv[i+1])
	}
	return o
}

func TestTallySourcesCountsATransitionOutAndOneBackPerRecord(t *testing.T) {
	null := "1969-12-31T18:00:00.000-06:00"
	rows := []*pyjson.Object{row("source", "a", "normal_time", "2026-11-17T10:00:00.000-06:00"),
		row("source", "b", "normal_time", null), row("source", "a", "normal_time", null),
		row("source", "b", "normal_time", ""), row("source", "b", "normal_time", null)}
	tallies := tallySources(rows)
	if len(tallies) != 2 || *tallies[0] != (sourceTally{"a", 2, 3}) || *tallies[1] != (sourceTally{"b", 3, 3}) {
		t.Fatalf("tallies %+v %+v", tallies[0], tallies[1])
	}
	if got := pyjson.Repr(chatteringSources(tallies, 3)); got != "[{'source': 'a', 'transitions': 3, 'records': 2}, "+
		"{'source': 'b', 'transitions': 3, 'records': 3}]" {
		t.Errorf("chattering at 3: %s", got)
	}
	if got := pyjson.Repr(chatteringSources(tallies, 4)); got != "[]" {
		t.Errorf("chattering at 4: %s", got)
	}
	if got := pyjson.Repr(busiestSources(tallies)); got != "[{'source': 'b', 'count': 3}, {'source': 'a', 'count': 2}]" {
		t.Errorf("busiest %s", got)
	}
}

func TestOpenFindingsSplitsOffnormalAndOldUnackedOldestFirst(t *testing.T) {
	now := time.Date(2026, 11, 17, 18, 0, 0, 0, time.UTC)
	rows := []*pyjson.Object{
		row("source", "new", "source_state", "offnormal", "ack_state", "unacked", "alarm_class", "c", "alarm_time", "2026-11-16T12:00:00.000-06:00"),
		row("source", "old", "source_state", "normal", "ack_state", "unacked", "alarm_class", "c", "alarm_time", "2026-11-10T12:00:00.000-06:00"),
		row("source", "acked", "source_state", "fault", "ack_state", "acked", "alarm_class", "c", "alarm_time", "2026-11-01T12:00:00.000-06:00"),
		row("source", "bad", "source_state", "normal", "ack_state", "unacked", "alarm_class", "c", "alarm_time", "yesterday"),
	}
	off, unacked, unreadable := openFindings(rows, now, 3)
	if got := pyjson.Repr(off); got != "[{'source': 'acked', 'source_state': 'fault', 'alarm_class': 'c', 'since': '2026-11-01T12:00:00.000-06:00'}, "+
		"{'source': 'new', 'source_state': 'offnormal', 'alarm_class': 'c', 'since': '2026-11-16T12:00:00.000-06:00'}]" {
		t.Errorf("offnormal %s", got)
	}
	if got := pyjson.Repr(unacked); got != "[{'source': 'old', 'alarm_class': 'c', 'ack_state': 'unacked', "+
		"'alarm_time': '2026-11-10T12:00:00.000-06:00', 'age_days': 7}]" || unreadable != 1 {
		t.Errorf("unacked %s, unreadable %d", got, unreadable)
	}
	if _, unacked, _ = openFindings(rows, now, 8); len(unacked) != 0 {
		t.Errorf("8 days: %v", unacked)
	}
}
