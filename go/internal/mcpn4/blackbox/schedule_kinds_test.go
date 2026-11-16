package blackbox

// n4_list_schedule_kinds: which schedule kinds and special-event kinds exist and their
// parameters, without a station session (schedules unit, slice 1).

import (
	"reflect"
	"testing"
)

func kindNames(rows []map[string]any) []any {
	out := []any{}
	for _, r := range rows {
		out = append(out, r["name"])
	}
	return out
}

func TestScheduleKindsListsEveryKindWithItsTypeWithoutASession(t *testing.T) {
	r := NewRig(t, nil) // never connected
	out := r.Srv.OK("n4_list_schedule_kinds", Args{})
	kinds := objects(out["schedule_kinds"])
	want := []any{"boolean", "numeric", "enum", "string", "calendar", "trigger"}
	if got := kindNames(kinds); !reflect.DeepEqual(got, want) {
		t.Fatalf("schedule kinds = %v", got)
	}
	if out["schedule_kind_count"] != float64(6) || out["event_kind_count"] != float64(5) {
		t.Errorf("counts = %v %v", out["schedule_kind_count"], out["event_kind_count"])
	}
	b := kinds[0]
	if b["type"] != "schedule:BooleanSchedule" || b["value_type"] != "baja:StatusBoolean" ||
		b["default_output"] != "false" || b["weekly"] != true || b["special_events"] != true {
		t.Errorf("boolean = %v", b)
	}
	if c := kinds[4]; c["type"] != "schedule:CalendarSchedule" || c["dates"] != true || c["weekly"] != false {
		t.Errorf("calendar = %v", c)
	}
	if tr := kinds[5]; tr["type"] != "schedule:TriggerSchedule" || tr["times"] != true || tr["value_type"] != nil {
		t.Errorf("trigger = %v", tr)
	}
	for _, k := range kinds {
		if k["purpose"] == "" || k["behaviour"] == "" || k["links"] == "" {
			t.Errorf("%v lacks a description: %v", k["name"], k)
		}
	}
}

func TestScheduleKindsListsTheFiveEventKindsWithTheirFields(t *testing.T) {
	r := NewRig(t, nil)
	events := objects(r.Srv.OK("n4_list_schedule_kinds", Args{})["event_kinds"])
	want := []any{"date", "date_range", "week_and_day", "custom", "reference"}
	if got := kindNames(events); !reflect.DeepEqual(got, want) {
		t.Fatalf("event kinds = %v", got)
	}
	fields := map[string][]any{}
	for _, e := range events {
		fields[e["name"].(string)] = kindNames(objects(e["fields"]))
		if e["in_calendar"] != (e["name"] != "reference") {
			t.Errorf("%v in_calendar = %v", e["name"], e["in_calendar"])
		}
		if e["type"] == "" || e["purpose"] == "" {
			t.Errorf("%v = %v", e["name"], e)
		}
	}
	if got := fields["date"]; !reflect.DeepEqual(got, []any{"year", "month", "day", "weekday"}) {
		t.Errorf("date fields = %v", got)
	}
	if got := fields["week_and_day"]; !reflect.DeepEqual(got, []any{"month", "week", "weekday"}) {
		t.Errorf("week_and_day fields = %v", got)
	}
	if got := fields["reference"]; !reflect.DeepEqual(got, []any{"calendar"}) {
		t.Errorf("reference fields = %v", got)
	}
}
