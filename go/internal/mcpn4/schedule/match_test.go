package schedule

import (
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func day(s string) time.Time {
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return d
}

func dateRule(year any, month string, d any, weekday string) *pyjson.Object {
	return obj("year", year, "month", month, "day", d, "weekday", weekday)
}

func TestMatchesEvaluatesEachRuleKindOnADay(t *testing.T) {
	cases := []struct {
		name, kind string
		rule       *pyjson.Object
		on         string
		want       bool
	}{
		{"date hit", "date", dateRule(2026, "nov", 15, "*"), "2026-11-15", true},
		{"date other year", "date", dateRule(2025, "nov", 15, "*"), "2026-11-15", false},
		{"date every year", "date", dateRule("*", "dec", 25, "*"), "2027-12-25", true},
		{"date odd months on the 1st", "date", dateRule("*", "odd", 1, "*"), "2026-03-01", true},
		{"date even months on the 1st", "date", dateRule("*", "even", 1, "*"), "2026-03-01", false},
		{"date weekday", "date", dateRule("*", "*", "*", "monday"), "2026-11-16", true},
		{"date day list", "date", dateRule("*", "jan", []any{1, 2}, "*"), "2026-01-02", true},
		{"thanksgiving", "week_and_day", obj("month", "nov", "week", "week4", "weekday", "thursday"), "2026-11-26", true},
		{"third thursday is not week4", "week_and_day", obj("month", "nov", "week", "week4", "weekday", "thursday"),
			"2026-11-19", false},
		{"week1 ends on the 7th", "week_and_day", obj("month", "*", "week", "week1", "weekday", "*"), "2026-11-07", true},
		{"week2 starts on the 8th", "week_and_day", obj("month", "*", "week", "week1", "weekday", "*"), "2026-11-08", false},
		{"last7days", "week_and_day", obj("month", "*", "week", "last7days", "weekday", "friday"), "2026-11-27", true},
		{"custom summer", "custom", obj("year", "*", "months", []any{"jun", "jul"}, "days", []any{}, "weeks", []any{},
			"weekdays", []any{"monday", "tuesday"}), "2026-07-07", true},
		{"custom summer, wrong weekday", "custom", obj("year", "*", "months", []any{"jun", "jul"}, "days", []any{},
			"weeks", []any{}, "weekdays", []any{"monday", "tuesday"}), "2026-07-08", false},
		{"range inside", "date_range", obj("start", dateRule(2026, "dec", 24, "*"), "end", dateRule(2026, "dec", 26, "*")),
			"2026-12-26", true},
		{"range after", "date_range", obj("start", dateRule(2026, "dec", 24, "*"), "end", dateRule(2026, "dec", 26, "*")),
			"2026-12-27", false},
		{"yearly range wraps", "date_range", obj("start", dateRule("*", "dec", 30, "*"), "end", dateRule("*", "jan", 2, "*")),
			"2027-01-01", true},
		{"yearly range wraps, outside", "date_range", obj("start", dateRule("*", "dec", 30, "*"),
			"end", dateRule("*", "jan", 2, "*")), "2027-01-03", false},
	}
	for _, c := range cases {
		got, ok := Matches(c.kind, c.rule, day(c.on))
		if !ok || got != c.want {
			t.Errorf("%s: Matches = %v ok %v, want %v", c.name, got, ok, c.want)
		}
	}
}

func TestMatchesNeverGuessesARuleItCannotEvaluate(t *testing.T) {
	cases := map[string]struct {
		kind string
		rule *pyjson.Object
	}{
		"calendar week":        {"week_and_day", obj("month", "nov", "week", "calendarWeek1", "weekday", "monday")},
		"week before the last": {"custom", obj("year", "*", "months", []any{}, "days", []any{}, "weeks", []any{"before_last7"}, "weekdays", []any{})},
		"wildcard month range": {"date_range", obj("start", dateRule(2026, "*", 1, "*"), "end", dateRule(2026, "*", 5, "*"))},
		"mixed range ends":     {"date_range", obj("start", dateRule(2026, "dec", 1, "*"), "end", dateRule("*", "dec", 5, "*"))},
		"reference":            {"reference", obj("calendar", "station:|slot:/Cal")},
		"missing rule":         {"date", nil},
	}
	for name, c := range cases {
		if _, ok := Matches(c.kind, c.rule, day("2026-11-20")); ok {
			t.Errorf("%s: evaluated, want not evaluated", name)
		}
	}
}

func TestFirstMatchFindsTheFirstDayInsideTheHorizonOnly(t *testing.T) {
	thanksgiving := obj("month", "nov", "week", "week4", "weekday", "thursday")
	d, found, ok := FirstMatch("week_and_day", thanksgiving, day("2026-11-20"), 7)
	if !ok || !found || d.Format("2006-01-02") != "2026-11-26" {
		t.Errorf("inside: %v %v %v", d, found, ok)
	}
	if _, found, ok := FirstMatch("week_and_day", thanksgiving, day("2026-11-20"), 6); !ok || found {
		t.Errorf("one day short of it: found %v ok %v", found, ok)
	}
	christmas := obj("start", dateRule(2026, "dec", 24, "*"), "end", dateRule(2026, "dec", 26, "*"))
	if _, found, ok := FirstMatch("date_range", christmas, day("2026-11-20"), 7); !ok || found {
		t.Errorf("christmas found %v ok %v", found, ok)
	}
}
