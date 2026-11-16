package schedule

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func ruleOf(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	v, err := pyjson.Unmarshal([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	return v.(*pyjson.Object)
}

// Each kind encodes the fields of n4_read_schedule's rule, as the station stores them.
func TestEncodeRuleBuildsEveryKind(t *testing.T) {
	sel := `{"n":"singleSelection","nm":"p","t":"baja:Boolean","v":"true"}`
	always := `{"n":"alwaysEffective","nm":"p","t":"baja:Boolean","v":"true"}`
	for _, c := range []struct{ kind, rule, want string }{
		{"date", `{"year": 2027, "month": "dec", "day": 24}`, `{"nm":"p","s":[{"n":"yearSchedule","nm":"p","s":[` +
			`{"n":"alwaysEffective","nm":"p","t":"baja:Boolean","v":"false"},{"n":"year","nm":"p","t":"baja:Integer","v":"2027"}],` +
			`"t":"schedule:YearSchedule"},{"n":"monthSchedule","nm":"p","s":[` + sel + `,{"n":"set","nm":"p","t":"baja:EnumSet","v":"11"}],` +
			`"t":"schedule:MonthSchedule"},{"n":"daySchedule","nm":"p","s":[` + sel + `,{"n":"set","nm":"p","t":"baja:EnumSet","v":"24"}],` +
			`"t":"schedule:DayOfMonthSchedule"},{"n":"weekdaySchedule","nm":"p","s":[` + sel + `],"t":"schedule:WeekdaySchedule"}],` +
			`"t":"schedule:DateSchedule"}`},
		{"week_and_day", `{"month": "*", "week": ["week1", "last7days"], "weekday": "friday"}`, `{"nm":"p","s":[` +
			`{"n":"months","nm":"p","s":[` + sel + `],"t":"schedule:MonthSchedule"},{"n":"weeksOfMonth","nm":"p","s":[` + sel +
			`,{"n":"set","nm":"p","t":"baja:EnumSet","v":"1,6"}],"t":"schedule:WeekOfMonthSchedule"},{"n":"weekdays","nm":"p","s":[` + sel +
			`,{"n":"set","nm":"p","t":"baja:EnumSet","v":"5"}],"t":"schedule:WeekdaySchedule"}],"t":"schedule:WeekAndDaySchedule"}`},
		{"custom", `{"months": ["jun", "odd"], "days": [1, 15], "weekdays": []}`, `{"nm":"p","s":[` +
			`{"n":"daysOfMonth","nm":"p","s":[{"n":"set","nm":"p","t":"baja:EnumSet","v":"1,15"}],"t":"schedule:DayOfMonthSchedule"},` +
			`{"n":"months","nm":"p","s":[{"n":"set","nm":"p","t":"baja:EnumSet","v":"5,12"}],"t":"schedule:MonthSchedule"},` +
			`{"n":"weekdays","nm":"p","s":[],"t":"schedule:WeekdaySchedule"},{"n":"weeksOfMonth","nm":"p","s":[],"t":"schedule:WeekOfMonthSchedule"},` +
			`{"n":"year","nm":"p","s":[` + always + `],"t":"schedule:YearSchedule"}],"t":"schedule:CustomSchedule"}`},
		{"reference", `{"calendar": "station:|slot:/Cal"}`, `{"nm":"p","s":[{"n":"ref","nm":"p","t":"baja:Ord","v":"station:|slot:/Cal"}],` +
			`"t":"schedule:ScheduleReference"}`},
	} {
		var f Faults
		got := EncodeRule(c.kind, ruleOf(t, c.rule), "rule", &f)
		if f.Err() != nil || compact(t, got) != c.want {
			t.Errorf("%s: %v\n%s\nwant\n%s", c.kind, f.Err(), compact(t, got), c.want)
		}
	}
	var f Faults
	got := compact(t, EncodeRule("date_range", ruleOf(t, `{"start": {"month": "jan"}, "end": {"year": 2030}}`), "r", &f))
	if f.Err() != nil || !strings.Contains(got, `"n":"start"`) || !strings.Contains(got, `"n":"end"`) ||
		!strings.Contains(got, `"v":"2030"`) || !strings.Contains(got, `"t":"schedule:DateRangeSchedule"`) {
		t.Errorf("date_range: %v %s", f.Err(), got)
	}
}

func TestEncodeRuleNamesEveryFault(t *testing.T) {
	var f Faults
	EncodeRule("date", ruleOf(t, `{"year": 1900, "month": "dic", "day": 32, "weekday": "lunes", "hour": 1}`), "r", &f)
	EncodeRule("week_and_day", ruleOf(t, `{"week": "*x"}`), "w", &f)
	EncodeRule("week_and_day", ruleOf(t, `{"week": ["*"]}`), "any", &f) // "*" alone means any, not in a list
	EncodeRule("reference", ruleOf(t, `{"calendar": "Cal"}`), "c", &f)
	EncodeRule("holiday", ruleOf(t, `{}`), "h", &f)
	EncodeRule("date_range", ruleOf(t, `{"start": 1}`), "d", &f)
	for _, w := range []string{`r: unknown field 'hour' (use year, month, day, weekday)`,
		`r.year: 1900 is not a year (1970-9999) or "*"`, `r.month: 'dic' is not a month (jan, feb,`,
		`r.day: 32 is not a day of the month (1-31)`, `r.weekday: 'lunes' is not a weekday (sunday,`,
		`w.week: '*x' is not a week (week1, week2,`, `any.week: '*' is not a week (week1,`, `c.calendar: 'Cal' is not a calendar's slot ORD`,
		`h: unknown kind 'holiday' (use date, date_range, week_and_day, custom or reference)`,
		`d.start: must be a date object`, `d.end: must be a date object`} {
		if !strings.Contains(f.Err().Error(), w) {
			t.Errorf("missing %q in\n%v", w, f.Err())
		}
	}
}
