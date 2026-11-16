package blackbox

// n4_set_schedule, weekly (schedules unit, slice 3): only the days given are replaced, every
// fault is named before anything is sent, the read-back compares the decoded schedule and
// n4_rollback restores the previous schedule exactly.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func scheduleWriteRig(t *testing.T) *Rig {
	return scheduleRig(t, true, func(r *Rig, o *Options) { r.WriteOptions(o) })
}

func setSchedule(name string, extra Args) Args {
	args := Args{"ord": Folder + "/" + name}
	for k, v := range extra {
		args[k] = v
	}
	return args
}

func TestSetScheduleReplacesOnlyTheDaysGivenAndReadsThemBack(t *testing.T) {
	r := scheduleWriteRig(t)
	before := readSchedule(r, "BoolSch")["weekly"].(map[string]any)
	weekly := map[string]any{
		"monday": []any{map[string]any{"start": "13:00", "end": "17:00", "value": false},
			map[string]any{"start": "08:00", "end": "12:00", "value": true}},
		"sunday": []any{map[string]any{"start": "00:00", "end": "24:00", "value": true}}}
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": weekly}))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	after := readSchedule(r, "BoolSch")["weekly"].(map[string]any)
	if got := jsonOf(t, after["monday"]); got != canon(t,
		`[{"start":"08:00","end":"12:00","value":true},{"start":"13:00","end":"17:00","value":false}]`) {
		t.Errorf("monday = %s", got)
	}
	if got := jsonOf(t, after["sunday"]); got != canon(t, `[{"start":"00:00","end":"24:00","value":true}]`) {
		t.Errorf("sunday = %s", got)
	}
	for _, day := range []string{"tuesday", "wednesday", "thursday", "friday", "saturday"} {
		if jsonOf(t, after[day]) != jsonOf(t, before[day]) {
			t.Errorf("%s changed: %s -> %s", day, jsonOf(t, before[day]), jsonOf(t, after[day]))
		}
	}
}

func TestSetScheduleClearsADayGivenAnEmptyList(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": map[string]any{"wednesday": []any{}}}))
	if got := jsonOf(t, readSchedule(r, "BoolSch")["weekly"].(map[string]any)["wednesday"]); got != "[]" {
		t.Errorf("wednesday = %s", got)
	}
}

func TestSetScheduleWritesTheValueOfEachKind(t *testing.T) {
	r := scheduleWriteRig(t)
	for name, value := range map[string]any{"NumSch": 19.5, "EnumSch": 3, "StrSch": "eco"} {
		args := setSchedule(name, Args{"weekly": map[string]any{"friday": []any{
			map[string]any{"start": "06:30", "end": "07:45", "value": value}}}})
		if out := r.Srv.RunWrite("n4_set_schedule", args); out["verdict"] != "verified" {
			t.Fatalf("%s: %v", name, out)
		}
		if s := jsonOf(t, readSchedule(r, name)["weekly"].(map[string]any)["friday"]); s != canon(t,
			`[{"start":"06:30","end":"07:45","value":`+jsonOf(t, value)+`}]`) {
			t.Errorf("%s friday = %s", name, s)
		}
	}
}

// Every fault is named (schedule.Weekly's own tests list each message) and nothing is sent.
func TestSetScheduleNamesEveryFaultAndSendsNothing(t *testing.T) {
	r := scheduleWriteRig(t)
	before := jsonOf(t, readSchedule(r, "BoolSch")["weekly"])
	weekly := map[string]any{"lunes": []any{}, "monday": []any{
		map[string]any{"start": "11:00", "end": "13:00", "value": "on"},
		map[string]any{"start": "12:00", "end": "14:00", "value": false}}}
	text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": weekly}))
	for _, want := range []string{"3 fault(s) in the request, nothing was sent", "weekly: unknown day 'lunes'",
		`weekly.monday[0]: value 'on' is not a boolean`, `weekly.monday: ranges 11:00-13:00 and 12:00-14:00 overlap`} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in:\n%s", want, text)
		}
	}
	if after := jsonOf(t, readSchedule(r, "BoolSch")["weekly"]); after != before {
		t.Errorf("the schedule changed: %s", after)
	}
}

func TestSetScheduleRefusesAWeeklyScheduleOnACalendarAndAnEmptyRequest(t *testing.T) {
	r := scheduleWriteRig(t)
	text := r.Srv.Err("n4_set_schedule", setSchedule("Cal", Args{"weekly": map[string]any{"monday": []any{}}}))
	if !strings.Contains(text, "weekly: a calendar schedule has no weekly schedule") {
		t.Errorf("calendar: %q", text)
	}
	if text := r.Srv.Err("n4_set_schedule", setSchedule("BoolSch", nil)); !strings.Contains(text, "nothing to set") {
		t.Errorf("empty: %q", text)
	}
}

func TestSetScheduleRollbackRestoresThePreviousWeekExactly(t *testing.T) {
	r := scheduleWriteRig(t)
	before := readSchedule(r, "StrSch")
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("StrSch", Args{"weekly": map[string]any{
		"monday": []any{}, "tuesday": []any{map[string]any{"start": "01:00", "end": "02:00", "value": "night"}}}}))
	if back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]}); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if after := readSchedule(r, "StrSch"); jsonOf(t, after["weekly"]) != jsonOf(t, before["weekly"]) {
		t.Errorf("weekly not restored: %s -> %s", jsonOf(t, before["weekly"]), jsonOf(t, after["weekly"]))
	}
}

func TestSetScheduleIsListedAsAnIdempotentNonDestructiveWrite(t *testing.T) {
	ann := annotations(t, listedTools(scheduleWriteRig(t).Srv)["n4_set_schedule"], "n4_set_schedule")
	if ann["readOnlyHint"] != false || ann["destructiveHint"] != false || ann["idempotentHint"] != true {
		t.Errorf("annotations %v", ann)
	}
}

// A day the station did not store as sent is reported, not raised: verdict mismatch.
func TestSetScheduleReportsADayTheStationDidNotKeep(t *testing.T) {
	r := scheduleWriteRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		day := st.Root().Child("Folder").Child("BoolSch").Child("schedule").Child("week").Child("monday").Child("day")
		day.Children = nil
	}})
	out := r.Srv.RunWrite("n4_set_schedule", setSchedule("BoolSch", Args{"weekly": map[string]any{"monday": []any{
		map[string]any{"start": "05:00", "end": "06:00", "value": true}}}}))
	if out["verdict"] != "mismatch" || jsonOf(t, out["observed"]) != canon(t, `{"weekly":{"monday":[]}}`) {
		t.Errorf("out = %v", out)
	}
}

// A schedule that is not there is named, with every fault that needs no station.
func TestSetScheduleNamesAMissingScheduleWithTheRequestFaults(t *testing.T) {
	r := scheduleWriteRig(t)
	text := r.Srv.Err("n4_set_schedule", setSchedule("Nope", Args{"weekly": map[string]any{"lunes": []any{},
		"monday": []any{map[string]any{"start": "09:00", "end": "08:00", "value": "x"}}}}))
	for _, want := range []string{"3 fault(s)", "weekly: unknown day 'lunes'", "weekly.monday[0]: end '08:00' is not after start '09:00'",
		"ord: no component at " + Folder + "/Nope ("} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q in %q", want, text)
		}
	}
	if strings.Contains(text, "is not a boolean") {
		t.Errorf("a value was checked without the schedule: %q", text)
	}
}
