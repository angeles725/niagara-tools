package schedule

import (
	"strings"
	"testing"
)

func TestDatesParsesEachCalendarKindAndNamesEveryFault(t *testing.T) {
	var f Faults
	ok := Dates(eventList(t, `[{"name": "New Year", "kind": "date", "rule": {"month": "jan", "day": 1}},
		{"name": "Fridays", "kind": "week_and_day", "rule": {"weekday": "friday"}}]`), eventDay, false, &f)
	if f.Err() != nil || len(ok) != 2 || ok[0].Slot != "New$20Year" || ok[1].Kind != "week_and_day" ||
		!strings.Contains(compact(t, ok[0].Body), `"t":"schedule:DateSchedule"`) {
		t.Fatalf("%v %v", f.Err(), ok)
	}
	var bad Faults
	Dates(eventList(t, `[{"name": "Ref", "kind": "reference", "rule": {"calendar": "station:|slot:/C"}},
		{"name": "Gone", "kind": "date", "rule": {"year": 2020}}, {"kind": "date", "x": 1}, "d"]`), eventDay, false, &bad)
	for _, w := range []string{"5 fault(s)",
		`add_date[0]: kind 'reference' is not a calendar date kind (use date, date_range, week_and_day, custom)`,
		`add_date[1]: 'Gone' can never occur again (it ends before 2026-11-15); pass allow_expired: true to add it anyway`,
		`add_date[2]: unknown field 'x' (use name, kind, rule)`, `add_date[2].name: must be a non-empty string`,
		`add_date[3]: must be an object {name, kind, rule}`} {
		if !strings.Contains(bad.Err().Error(), w) {
			t.Errorf("missing %q in\n%v", w, bad.Err())
		}
	}
}

// A requested rule compares with the rule n4_read_schedule decodes.
func TestRuleViewIsTheReadShape(t *testing.T) {
	for _, c := range []struct{ kind, rule, want string }{
		{"date", `{"month": "jan", "day": 1, "weekday": []}`, `{"day":1,"month":"jan","weekday":"*","year":"*"}`},
		{"week_and_day", `{"week": ["week2"], "weekday": ["monday", "friday"]}`, `{"month":"*","week":"week2","weekday":["monday","friday"]}`},
		{"custom", `{"months": "jun", "weekdays": "*"}`, `{"days":[],"months":["jun"],"weekdays":[],"weeks":[],"year":"*"}`},
		{"date_range", `{"end": {"year": 2027}}`, `{"end":{"day":"*","month":"*","weekday":"*","year":2027},` +
			`"start":{"day":"*","month":"*","weekday":"*","year":"*"}}`},
	} {
		if got := compact(t, RuleView(c.kind, ruleOf(t, c.rule))); got != c.want {
			t.Errorf("%s: %s, want %s", c.kind, got, c.want)
		}
	}
}

func TestTriggerTimesAreSortedTwentySecondRanges(t *testing.T) {
	var f Faults
	times := TriggerTimes(eventList(t, `["18:30", "06:00"]`), &f)
	if f.Err() != nil || len(times) != 2 || times[0] != 360 {
		t.Fatalf("%v %v", f.Err(), times)
	}
	want := `{"nm":"p","s":[{"n":"time","nm":"p","s":[{"n":"start","nm":"p","t":"baja:Time","v":"06:00:00.000"},` +
		`{"n":"finish","nm":"p","t":"baja:Time","v":"06:00:20.000"}],"t":"schedule:TimeSchedule"},{"n":"time1","nm":"p","s":[` +
		`{"n":"start","nm":"p","t":"baja:Time","v":"18:30:00.000"},{"n":"finish","nm":"p","t":"baja:Time","v":"18:30:20.000"}],` +
		`"t":"schedule:TimeSchedule"}],"t":"schedule:DaySchedule"}`
	if got := compact(t, TimesBody(times)); got != want {
		t.Errorf("body =\n%s\nwant\n%s", got, want)
	}
	var bad Faults
	TriggerTimes(eventList(t, `["06:00", "6:00", "06:00", 7]`), &bad)
	for _, w := range []string{"3 fault(s)", "times[1]: '6:00' is not a time HH:MM from 00:00 to 23:59",
		"times[2]: '06:00' is given twice", "times[3]: 7 is not a time"} {
		if !strings.Contains(bad.Err().Error(), w) {
			t.Errorf("missing %q in\n%v", w, bad.Err())
		}
	}
}
