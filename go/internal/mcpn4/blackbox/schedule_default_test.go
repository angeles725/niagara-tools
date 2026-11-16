package blackbox

// n4_set_schedule, default output (schedules unit, slice 3): the value out takes when nothing
// is active, of each kind or null, checked against the kind and rolled back exactly.

import (
	"strings"
	"testing"
)

func TestSetScheduleWritesTheDefaultOutputOfEachKind(t *testing.T) {
	r := scheduleWriteRig(t)
	for name, value := range map[string]any{"BoolSch": true, "NumSch": 19.5, "EnumSch": 3, "StrSch": "eco", "Cal": true} {
		if out := r.Srv.RunWrite("n4_set_schedule", setSchedule(name, Args{"default_output": value})); out["verdict"] != "verified" {
			t.Fatalf("%s: %v", name, out)
		}
		if s := jsonOf(t, readSchedule(r, name)["default_output"]); s != canon(t, `{"value":`+jsonOf(t, value)+`,"status":"ok"}`) {
			t.Errorf("%s default_output = %s", name, s)
		}
	}
	r.Srv.RunWrite("n4_set_schedule", setSchedule("NumSch", Args{"default_output": nil}))
	if s := jsonOf(t, readSchedule(r, "NumSch")["default_output"]); s != canon(t, `{"value":null,"status":"null"}`) {
		t.Errorf("null default_output = %s", s)
	}
}

func TestSetScheduleRefusesADefaultOutputOfTheWrongTypeOrKind(t *testing.T) {
	r := scheduleWriteRig(t)
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"default_output": 5,
		"weekly": map[string]any{"lunes": []any{}}}))
	for _, want := range []string{"2 fault(s)", "weekly: unknown day 'lunes'",
		"default_output: value 5 is not a boolean (a boolean schedule takes true or false)"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if text := r.Srv.Err("n4_set_schedule", setSchedule("Cal", Args{"default_output": "x"})); !strings.Contains(
		text, "default_output: value 'x' is not a boolean (a calendar schedule takes true or false)") {
		t.Errorf("calendar: %q", text)
	}
	if text := r.Srv.Err("n4_set_schedule", setSchedule("TrigSch", Args{"default_output": true})); !strings.Contains(
		text, "default_output: a trigger schedule has no default output") {
		t.Errorf("trigger: %q", text)
	}
}

// The rollback restores the default output and the days of every kind, an omitted
// defaultOutput (StrSch's, at its null default) included.
func TestSetScheduleRollbackRestoresTheDefaultOutputExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	for name, v := range map[string][2]any{"NumSch": {12, 7.25}, "EnumSch": {1, 3}, "StrSch": {"x", "night"}} {
		before := readSchedule(r, name)
		out := r.Srv.RunWrite("n4_set_schedule", setSchedule(name, Args{"default_output": v[0],
			"weekly": map[string]any{"tuesday": []any{map[string]any{"start": "01:00", "end": "02:00", "value": v[1]}}}}))
		if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
			t.Errorf("%s rollback = %v", name, back)
		}
		after := readSchedule(r, name)
		for _, k := range []string{"weekly", "default_output"} {
			if jsonOf(t, after[k]) != jsonOf(t, before[k]) {
				t.Errorf("%s %s not restored: %s -> %s", name, k, jsonOf(t, before[k]), jsonOf(t, after[k]))
			}
		}
	}
}
