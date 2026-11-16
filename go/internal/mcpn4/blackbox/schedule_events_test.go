package blackbox

// n4_set_schedule, special events (schedules unit, slice 4): the specialEvents list is replaced
// whole in one set (the order is the priority), checked first, read back and rolled back.

import (
	"strings"
	"testing"
)

func eventNames(t *testing.T, r *Rig, name string) string {
	t.Helper()
	names := []string{}
	for _, e := range objects(readSchedule(r, name)["special_events"]) {
		names = append(names, e["name"].(string))
	}
	return strings.Join(names, ",")
}

var shutdown = map[string]any{"name": "Plant Shutdown", "kind": "date_range",
	"rule":   map[string]any{"start": map[string]any{"year": 2099, "month": "dec", "day": 24}, "end": map[string]any{"year": 2099, "month": "dec", "day": 31}},
	"ranges": []any{map[string]any{"start": "00:00", "end": "24:00", "value": false}}}

func TestSetScheduleAddsAndRemovesSpecialEventsKeepingTheOrder(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{shutdown},
		"remove_special_event": []any{"Christmas Break", "Probe"}}))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	if got := eventNames(t, r, "BoolSch"); got != "Thanksgiving,Summer,Holidays,Plant Shutdown" {
		t.Errorf("events = %s", got)
	}
	added := objects(readSchedule(r, "BoolSch")["special_events"])[3]
	if got := jsonOf(t, added); got != canon(t, `{"name":"Plant Shutdown","kind":"date_range","rule":{"start":{"year":2099,"month":"dec",`+
		`"day":24,"weekday":"*"},"end":{"year":2099,"month":"dec","day":31,"weekday":"*"}},"priority":4,`+
		`"ranges":[{"start":"00:00","end":"24:00","value":false}]}`) {
		t.Errorf("added = %s", got)
	}
}

func TestSetScheduleRefusesAnExpiredEventUnlessAllowed(t *testing.T) {
	r := scheduleWriteRig(t)
	old := map[string]any{"name": "Old", "kind": "date", "rule": map[string]any{"year": 2020, "month": "jan", "day": 1}, "ranges": []any{}}
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{old}}))
	if !strings.Contains(text, "add_special_event[0]: 'Old' can never occur again (it ends before ") ||
		!strings.Contains(text, "pass allow_expired: true to add it anyway") {
		t.Errorf("expired: %q", text)
	}
	r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"add_special_event": []any{old}, "allow_expired": true}))
	if got := eventNames(t, r, "BoolSch"); !strings.HasSuffix(got, ",Old") {
		t.Errorf("events = %s", got)
	}
}

func TestSetScheduleNamesEveryEventFault(t *testing.T) {
	r := scheduleWriteRig(t)
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"remove_special_event": []any{"Nope"},
		"add_special_event": []any{map[string]any{"name": "Summer", "kind": "date", "rule": map[string]any{}, "ranges": []any{}}}}))
	for _, want := range []string{"2 fault(s)", `remove_special_event: no event is named 'Nope'`,
		`add_special_event: 'Summer' already exists; remove it in the same call to replace it`} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if text := r.Srv.Err("n4_set_schedule", setSchedule("Cal", Args{"remove_special_event": []any{"x"}})); !strings.Contains(
		text, "special events: a calendar schedule has none") {
		t.Errorf("calendar: %q", text)
	}
}

func TestSetScheduleReplacesAnEventAndRollsBackTheList(t *testing.T) {
	r := scheduleWriteRig(t)
	before := jsonOf(t, readSchedule(r, "BoolSch")["special_events"])
	summer := map[string]any{"name": "Summer", "kind": "custom", "rule": map[string]any{"months": []any{"jul", "aug"}},
		"ranges": []any{map[string]any{"start": "10:00", "end": "11:00", "value": true}}}
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"remove_special_event": []any{"Summer"},
		"add_special_event": []any{summer}}))
	if got := eventNames(t, r, "BoolSch"); got != "Probe,Christmas Break,Thanksgiving,Holidays,Summer" {
		t.Errorf("events = %s", got)
	}
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if after := jsonOf(t, readSchedule(r, "BoolSch")["special_events"]); after != before {
		t.Errorf("not restored:\n%s\n%s", before, after)
	}
}
