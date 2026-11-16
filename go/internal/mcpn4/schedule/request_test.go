package schedule

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func weeklyArg(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

func TestWeeklySortsTheRangesOfEachDayGiven(t *testing.T) {
	var f Faults
	days := Weekly(weeklyArg(t, `{"monday": [{"start": "13:00", "end": "24:00", "value": false},
		{"start": "08:00", "end": "13:00", "value": true}], "sunday": []}`), KindOf("schedule:BooleanSchedule"), "", &f)
	if f.Err() != nil {
		t.Fatal(f.Err())
	}
	if m := days["monday"]; len(m) != 2 || m[0] != (Range{480, 780, true}) || m[1] != (Range{780, 1440, false}) {
		t.Errorf("monday = %v", m)
	}
	if s, ok := days["sunday"]; !ok || len(s) != 0 || len(days) != 2 {
		t.Errorf("days = %v", days)
	}
}

func TestWeeklyNamesEveryFault(t *testing.T) {
	var f Faults
	Weekly(weeklyArg(t, `{"lunes": [], "tuesday": {}, "friday": [{"start": "15:00", "end": "15:00", "value": true},
		{"start": "24:00", "end": "24:00", "value": true}], "monday": [
		{"start": "8:00", "end": "09:00", "value": true},
		{"start": "10:00", "end": "09:30", "value": true},
		{"start": "11:00", "end": "13:00", "value": "on"},
		{"start": "12:00", "end": "14:00", "value": false},
		{"start": "20:00", "end": "24:01", "value": true},
		{"start": "21:00", "end": "22:00"}, 7]}`), KindOf("schedule:BooleanSchedule"), "", &f)
	want := []string{
		`weekly: unknown day 'lunes' (use monday, tuesday, wednesday, thursday, friday, saturday, sunday)`,
		`weekly.tuesday: must be a list of {start, end, value}`,
		`weekly.monday[0]: start '8:00' is not a time HH:MM from 00:00 to 23:59`,
		`weekly.monday[1]: end '09:30' is not after start '10:00'`,
		`weekly.monday[2]: value 'on' is not a boolean (a boolean schedule takes true or false)`,
		`weekly.monday[4]: end '24:01' is not a time HH:MM from 00:01 to 24:00`,
		`weekly.monday[5]: value is missing`,
		`weekly.monday[6]: must be an object {start, end, value}`,
		`weekly.monday: ranges 11:00-13:00 and 12:00-14:00 overlap`,
		`weekly.friday[0]: end '15:00' is not after start '15:00'`,
		`weekly.friday[1]: start '24:00' is not a time HH:MM from 00:00 to 23:59`,
	}
	text := f.Err().Error()
	if !strings.HasPrefix(text, "11 fault(s) in the request, nothing was sent:") {
		t.Errorf("head: %q", text)
	}
	for _, w := range want {
		if !strings.Contains(text, "\n- "+w) {
			t.Errorf("missing %q in\n%s", w, text)
		}
	}
}

func TestCheckValueFollowsTheKind(t *testing.T) {
	kind := func(n string) *Kind { return KindOf("schedule:" + n) }
	num := func(s string) any { v, _ := pyjson.Unmarshal([]byte(s)); return v }
	for _, c := range []struct {
		k      *Kind
		facets string
		value  any
		null   bool
		want   string
	}{
		{kind("BooleanSchedule"), "", true, false, ""},
		{kind("NumericSchedule"), "", num("19.5"), false, ""},
		{kind("NumericSchedule"), "", "19.5", false, "value '19.5' is not a number (a numeric schedule takes a number)"},
		{kind("StringSchedule"), "", num("3"), false, "value 3 is not a string (a string schedule takes a string)"},
		{kind("EnumSchedule"), "", num("3"), false, ""},
		{kind("EnumSchedule"), "", num("1.5"), false, "value 1.5 is not an enum ordinal (an enum schedule takes a whole number >= 0)"},
		{kind("EnumSchedule"), "", num("-1"), false, "value -1 is not an enum ordinal (an enum schedule takes a whole number >= 0)"},
		{kind("EnumSchedule"), "range=E:{off=0,on=1}", num("2"), false, "value 2 is not an ordinal of the schedule's range facet (E:{off=0,on=1})"},
		{kind("EnumSchedule"), "range=E:{off=0,on=1}", num("1"), false, ""},
		{kind("BooleanSchedule"), "", nil, true, ""},
		{kind("BooleanSchedule"), "", nil, false, "value null is not allowed for a range (only a default output may be null)"},
	} {
		if got := CheckValue(c.k, c.facets, c.value, c.null); got != c.want {
			t.Errorf("%s %v: %q, want %q", c.k.Name, c.value, got, c.want)
		}
	}
}
