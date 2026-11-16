package blackbox

// n4_set_schedule, calendar dates and trigger times (schedules unit, slice 4): a date is removed
// with one remove and added bare then set by path, a trigger's times are one set; all of it is
// checked first, read back and rolled back exactly.

import (
	"strings"
	"testing"
)

var fridays = map[string]any{"name": "Fridays", "kind": "week_and_day", "rule": map[string]any{"weekday": "friday"}}

func TestSetScheduleAddsAndRemovesCalendarDates(t *testing.T) {
	r := scheduleWriteRig(t)
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("Cal", Args{"add_date": []any{fridays}, "remove_date": []any{"NewYear"}}))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	want := `[{"name":"Shutdown","kind":"date_range","rule":{"start":{"year":2026,"month":"dec","day":28,"weekday":"*"},` +
		`"end":{"year":2027,"month":"jan","day":2,"weekday":"*"}}},` +
		`{"name":"Fridays","kind":"week_and_day","rule":{"month":"*","week":"*","weekday":"friday"}}]`
	if got := jsonOf(t, readSchedule(r, "Cal")["dates"]); got != canon(t, want) {
		t.Errorf("dates =\n%s\nwant\n%s", got, want)
	}
	ops := planOf(r.Srv.Dry("n4_set_schedule", setSchedule("Cal", Args{"add_date": []any{
		map[string]any{"name": "Day", "kind": "date", "rule": map[string]any{"month": "may", "day": 1}}}})))["ops"].([]any)
	if len(ops) != 5 || ops[0].(map[string]any)["nm"] != "a" || ops[1].(map[string]any)["n"] != "Day/yearSchedule" {
		t.Errorf("an added date is a bare add then one set per frozen child: %v", ops)
	}
}

func TestSetScheduleRollsBackARemovedDateExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	before := jsonOf(t, readSchedule(r, "Cal")["dates"])
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("Cal", Args{"remove_date": []any{"NewYear", "Shutdown"},
		"add_date": []any{map[string]any{"name": "January", "kind": "custom", "rule": map[string]any{"months": []any{"jan"}}}}}))
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if after := jsonOf(t, readSchedule(r, "Cal")["dates"]); after != before {
		t.Errorf("dates not restored:\n%s\n%s", before, after)
	}
}

func TestSetScheduleNamesEveryDateFault(t *testing.T) {
	r := scheduleWriteRig(t)
	text := r.Srv.Err("n4_set_schedule", setSchedule("Cal", Args{"remove_date": []any{"Easter", "NewYear"},
		"add_date": []any{map[string]any{"name": "Shutdown", "kind": "date", "rule": map[string]any{}},
			map[string]any{"name": "Ref", "kind": "reference", "rule": map[string]any{"calendar": "station:|slot:/C"}},
			map[string]any{"name": "NewYear", "kind": "date", "rule": map[string]any{}}}}))
	for _, want := range []string{"4 fault(s)", `remove_date: no date is named 'Easter'`, `add_date: 'Shutdown' already exists`,
		`add_date: 'NewYear' is also removed in this call; replace a date in two calls (remove_date, then add_date)`,
		`add_date[1]: kind 'reference' is not a calendar date kind`} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	for name, args := range map[string]Args{"BoolSch": {"add_date": []any{fridays}}, "Cal": {"times": []any{"06:00"}}} {
		text := r.Srv.Err("n4_set_schedule", setSchedule(name, args))
		if !strings.Contains(text, "schedule has none") {
			t.Errorf("%s: %q", name, text)
		}
	}
}

func TestSetScheduleChangesATriggersDatesAndTimes(t *testing.T) {
	r := scheduleWriteRig(t)
	before := readSchedule(r, "TrigSch")
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("TrigSch", Args{"times": []any{"21:15", "07:30", "12:00"},
		"add_date": []any{fridays}, "remove_date": []any{"Weekdays"}}))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	after := readSchedule(r, "TrigSch")
	if got := jsonOf(t, after["times"]); got != `["07:30","12:00","21:15"]` {
		t.Errorf("times = %s", got)
	}
	if got := jsonOf(t, after["dates"]); !strings.Contains(got, `"name":"Fridays"`) || strings.Contains(got, "Weekdays") {
		t.Errorf("dates = %s", got)
	}
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	restored := readSchedule(r, "TrigSch")
	for _, k := range []string{"times", "dates"} {
		if jsonOf(t, restored[k]) != jsonOf(t, before[k]) {
			t.Errorf("%s not restored: %s -> %s", k, jsonOf(t, before[k]), jsonOf(t, restored[k]))
		}
	}
}
