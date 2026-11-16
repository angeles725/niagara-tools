package blackbox

// n4_read_schedule over the six schedules recorded live (schedules unit, slice 2): every kind
// decodes, omitted slots are filled from the type registry and never guessed.

import (
	"strconv"
	"strings"
	"testing"
)

func TestReadScheduleDecodesTheWeeklyRangesOfABooleanSchedule(t *testing.T) {
	r := scheduleRig(t, true, nil)
	out := readSchedule(r, "BoolSch")
	if out["kind"] != "boolean" || out["type"] != "schedule:BooleanSchedule" || out["effective"] != "always" {
		t.Errorf("head = %v %v %v", out["kind"], out["type"], out["effective"])
	}
	want := `{"monday":[{"start":"08:00","end":"18:00","value":null,"ignored":true}],` +
		`"tuesday":[{"start":"08:00","end":"18:00","value":true}],` +
		`"wednesday":[{"start":"07:00","end":"12:00","value":true},{"start":"13:00","end":"24:00","value":true}],` +
		`"thursday":[],"friday":[],"saturday":[],"sunday":[]}`
	if got := jsonOf(t, out["weekly"]); got != canon(t, want) {
		t.Errorf("weekly =\n%s\nwant\n%s", got, want)
	}
}

func TestReadScheduleReportsOutAndWhatDrivesIt(t *testing.T) {
	r := scheduleRig(t, true, nil)
	out := readSchedule(r, "BoolSch")
	checks := map[string]string{"out": `{"value":true,"status":"ok"}`, "out_source": `"Special Event: Probe"`,
		"next_time": `"2026-11-16T00:00:00.000-06:00"`, "next_value": `{"value":false,"status":"ok"}`,
		"default_output": `{"value":false,"status":"ok"}`, "in": `{"value":null,"status":"null"}`,
		"last_modified": `"2026-11-15T22:13:47.960-06:00"`}
	for k, want := range checks {
		if got := jsonOf(t, out[k]); got != canon(t, want) {
			t.Errorf("%s = %s, want %s", k, got, want)
		}
	}
}

func TestReadScheduleDecodesEverySpecialEventKindInPriorityOrder(t *testing.T) {
	r := scheduleRig(t, true, nil)
	events := objects(readSchedule(r, "BoolSch")["special_events"])
	want := []string{
		`{"name":"Probe","kind":"date","rule":{"year":2026,"month":"nov","day":15,"weekday":"*"},"priority":1,` +
			`"ranges":[{"start":"00:00","end":"24:00","value":true}]}`,
		`{"name":"Christmas Break","kind":"date_range","rule":{"start":{"year":2026,"month":"dec","day":24,` +
			`"weekday":"*"},"end":{"year":2026,"month":"dec","day":26,"weekday":"*"}},"priority":2,` +
			`"ranges":[{"start":"00:00","end":"24:00","value":false}]}`,
		`{"name":"Thanksgiving","kind":"week_and_day","rule":{"month":"nov","week":"week4","weekday":"thursday"},` +
			`"priority":3,"ranges":[{"start":"06:00","end":"12:30","value":true}]}`,
		`{"name":"Summer","kind":"custom","rule":{"year":"*","months":["jun","jul"],"days":[],"weeks":[],` +
			`"weekdays":["monday","tuesday"]},"priority":4,"ranges":[{"start":"07:00","end":"09:00","value":true}]}`,
		`{"name":"Holidays","kind":"reference","rule":{"calendar":"station:|slot:/ImplSCH/Cal"},"priority":5,` +
			`"ranges":[]}`,
	}
	if len(events) != len(want) {
		t.Fatalf("%d events: %s", len(events), jsonOf(t, events))
	}
	for i, w := range want {
		if got := jsonOf(t, events[i]); got != canon(t, w) {
			t.Errorf("event %d =\n%s\nwant\n%s", i, got, w)
		}
	}
}

func TestReadScheduleDecodesTheValueOfEachWeeklyKind(t *testing.T) {
	r := scheduleRig(t, true, nil)
	for name, want := range map[string]string{
		"NumSch":  `{"start":"09:00","end":"17:00","value":21.5}`,
		"EnumSch": `{"start":"09:00","end":"17:00","value":2}`,
		"StrSch":  `{"start":"09:00","end":"17:00","value":"hello"}`,
	} {
		out := readSchedule(r, name)
		monday := objects(out["weekly"].(map[string]any)["monday"])
		if len(monday) != 1 || jsonOf(t, monday[0]) != canon(t, want) {
			t.Errorf("%s monday = %s", name, jsonOf(t, monday))
		}
		if got := jsonOf(t, out["out"]); !strings.Contains(got, `"status":"null"`) {
			t.Errorf("%s out = %s", name, got)
		}
	}
	// StringSchedule leaves its defaultOutput out (null status): it comes from the registry.
	if got := jsonOf(t, readSchedule(r, "StrSch")["default_output"]); got != canon(t, `{"value":null,"status":"null"}`) {
		t.Errorf("StrSch default_output = %s", got)
	}
}

func TestReadScheduleListsTheDatesOfACalendar(t *testing.T) {
	r := scheduleRig(t, true, nil)
	out := readSchedule(r, "Cal")
	want := `[{"name":"NewYear","kind":"date","rule":{"year":"*","month":"jan","day":1,"weekday":"*"}},` +
		`{"name":"Shutdown","kind":"date_range","rule":{"start":{"year":2026,"month":"dec","day":28,"weekday":"*"},` +
		`"end":{"year":2027,"month":"jan","day":2,"weekday":"*"}}}]`
	if got := jsonOf(t, out["dates"]); got != canon(t, want) {
		t.Errorf("dates =\n%s\nwant\n%s", got, want)
	}
	if out["kind"] != "calendar" || out["weekly"] != nil || out["special_events"] != nil || out["out_source"] != nil {
		t.Errorf("calendar reply = %v", out)
	}
}

func TestReadScheduleListsTheTimesAndDatesOfATrigger(t *testing.T) {
	r := scheduleRig(t, true, nil)
	out := readSchedule(r, "TrigSch")
	if got := jsonOf(t, out["times"]); got != `["06:00","18:30"]` {
		t.Errorf("times = %s", got)
	}
	if got := jsonOf(t, out["dates"]); got != canon(t, `[{"name":"Weekdays","kind":"week_and_day","rule":`+
		`{"month":"*","week":"*","weekday":"monday"}}]`) {
		t.Errorf("dates = %s", got)
	}
	if out["enabled"] != true || out["next_trigger"] != "2026-11-16T06:00:00.000-06:00" || out["last_trigger"] != nil ||
		out["out"] != nil {
		t.Errorf("trigger = %v", out)
	}
}

func TestReadScheduleRefusesWhatIsNoScheduleAndAnUndescribedOmittedSlot(t *testing.T) {
	r := scheduleRig(t, true, nil)
	r.Add("3", "Plain", "baja:Folder")
	if text := r.Srv.Err("n4_read_schedule", Args{"ord": Folder + "/Plain"}); !strings.Contains(text,
		"baja:Folder is not a schedule kind; n4_list_schedule_kinds lists them") {
		t.Errorf("folder: %q", text)
	}
	bare := scheduleRig(t, false, nil)
	text := bare.Srv.Err("n4_read_schedule", Args{"ord": Folder + "/BoolSch"})
	if !strings.Contains(text, "and its type registry does not describe that slot") {
		t.Errorf("no registry: %q", text)
	}
}

func TestReadScheduleSortsRangesKeepsSecondsAndShowsAFaultStatus(t *testing.T) {
	r := scheduleRig(t, true, nil)
	r.Set(handleAt(r, "BoolSch", "schedule/week/thursday"), "day", daySchedule(
		timeRange("time", "13:00:00.000", "17:30:15.000", "baja:StatusBoolean", "true", "2"),
		timeRange("time1", "07:00:00.000", "12:00:00.000", "baja:StatusBoolean", "false", "")))
	want := `[{"start":"07:00","end":"12:00","value":false},{"start":"13:00","end":"17:30:15","value":true,"status":"2"}]`
	if got := jsonOf(t, readSchedule(r, "BoolSch")["weekly"].(map[string]any)["thursday"]); got != canon(t, want) {
		t.Errorf("thursday = %s", got)
	}
}

func TestReadScheduleReportsALimitedEffectivePeriod(t *testing.T) {
	r := scheduleRig(t, true, nil)
	date := func(n string, month int) map[string]any {
		part := func(slot, typ string, kids ...any) map[string]any {
			return map[string]any{"nm": "p", "n": slot, "t": typ, "s": kids}
		}
		return part(n, "schedule:DateSchedule",
			part("yearSchedule", "schedule:YearSchedule",
				map[string]any{"nm": "p", "n": "alwaysEffective", "t": "baja:Boolean", "v": "true"}),
			part("monthSchedule", "schedule:MonthSchedule",
				map[string]any{"nm": "p", "n": "set", "t": "baja:EnumSet", "v": strconv.Itoa(month)}),
			part("daySchedule", "schedule:DayOfMonthSchedule"), part("weekdaySchedule", "schedule:WeekdaySchedule"))
	}
	r.Set(handleAt(r, "BoolSch", "effective"), "start", date("start", 4))
	want := `{"start":{"year":"*","month":"may","day":"*","weekday":"*"},` +
		`"end":{"year":"*","month":"*","day":"*","weekday":"*"}}`
	if got := jsonOf(t, readSchedule(r, "BoolSch")["effective"]); got != canon(t, want) {
		t.Errorf("effective = %s", got)
	}
}
