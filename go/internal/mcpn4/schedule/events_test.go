package schedule

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func eventList(t *testing.T, text string) []any {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.([]any)
}

var eventDay = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

func TestSpecialEventsBuildsADailyScheduleOfTheDayAndItsRule(t *testing.T) {
	var f Faults
	k := KindOf("schedule:BooleanSchedule")
	events := SpecialEvents(eventList(t, `[{"name": "Plant Shutdown", "kind": "date", "rule": {"year": 2026, "month": "dec", "day": 24},
		"ranges": [{"start": "00:00", "end": "24:00", "value": false}]}]`), k, "", eventDay, false, &f)
	if f.Err() != nil || len(events) != 1 {
		t.Fatalf("%v %v", f.Err(), events)
	}
	got := compact(t, EventBody(k, events[0]))
	for _, w := range []string{`"n":"Plant$20Shutdown"`, `"t":"schedule:DailySchedule"`, `"n":"day"`, `"t":"schedule:DaySchedule"`,
		`"n":"days"`, `"t":"schedule:DateSchedule"`, `"v":"2026"`, `"f":"o1"`} {
		if !strings.Contains(got, w) {
			t.Errorf("missing %s in %s", w, got)
		}
	}
	if events[0].Name != "Plant Shutdown" {
		t.Errorf("name = %q", events[0].Name)
	}
}

func TestSpecialEventsRefusesAnExpiredDateUnlessAllowed(t *testing.T) {
	k := KindOf("schedule:BooleanSchedule")
	list := eventList(t, `[{"name": "Old", "kind": "date_range", "rule": {"start": {"year": 2025}, "end": {"year": 2025}}, "ranges": []},
		{"name": "", "kind": "date", "rule": {}, "ranges": {}, "colour": 1}, 3]`)
	var f Faults
	SpecialEvents(list, k, "", eventDay, false, &f)
	for _, w := range []string{"5 fault(s)",
		`add_special_event[0]: 'Old' can never occur again (it ends before 2026-11-15); pass allow_expired: true to add it anyway`,
		`add_special_event[1]: unknown field 'colour' (use name, kind, rule, ranges)`, `add_special_event[1].name: must be a non-empty string`,
		`add_special_event[1].ranges: must be a list of {start, end, value}`, `add_special_event[2]: must be an object {name, kind, rule, ranges}`} {
		if !strings.Contains(f.Err().Error(), w) {
			t.Errorf("missing %q in\n%v", w, f.Err())
		}
	}
	var allowed Faults
	SpecialEvents(list[:1], k, "", eventDay, true, &allowed)
	if allowed.Err() != nil {
		t.Errorf("allow_expired: %v", allowed.Err())
	}
}

func TestExpiredOnlyForADateThatCannotComeAgain(t *testing.T) {
	today := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	for rule, want := range map[string]bool{
		`{"year": 2025}`: true, `{"year": 2026, "month": "oct"}`: true, `{"year": 2026, "month": "nov", "day": 14}`: true,
		`{"year": 2026, "month": "nov", "day": 15}`: false, `{"year": 2026, "month": "nov"}`: false, `{"year": 2026}`: false,
		`{"month": "jan", "day": 1}`: false, `{"year": 2026, "month": "odd"}`: false, `{"year": 2027, "month": "jan"}`: false,
	} {
		if got := Expired("date", ruleOf(t, rule), today); got != want {
			t.Errorf("date %s: %v", rule, got)
		}
	}
	if !Expired("date_range", ruleOf(t, `{"start": {"year": 2020}, "end": {"year": 2026, "month": "jan"}}`), today) ||
		Expired("date_range", ruleOf(t, `{"start": {"year": 2020}, "end": {"year": 2026, "month": "dec"}}`), today) ||
		Expired("week_and_day", ruleOf(t, `{"month": "jan"}`), today) {
		t.Error("date_range / week_and_day")
	}
}

func TestEscapeNameIsSlotPathEscaping(t *testing.T) {
	for in, want := range map[string]string{"Christmas Break": "Christmas$20Break", "1st": "$31st", "Año_2": "A$u00f1o_2", "a-b": "a$2db"} {
		if got := EscapeName(in); got != want {
			t.Errorf("%q = %q, want %q", in, got, want)
		}
	}
}
