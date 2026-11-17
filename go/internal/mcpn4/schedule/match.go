package schedule

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Day matching of decoded day rules (the plain rules Read returns), for the schedule audit's
// upcoming special events. A rule matches a day only when every field matches it; a field this
// matcher cannot evaluate (a calendar week, a week before the last, a wildcard date range) makes
// the whole rule not evaluated (ok false), never a guessed match.

// FirstMatch is the first day of [from, from+days) the rule of kind matches; found false when
// none does, ok false when the rule cannot be evaluated (a reference included: its days are a
// calendar's, which the caller resolves).
func FirstMatch(kind string, rule *pyjson.Object, from time.Time, days int) (day time.Time, found, ok bool) {
	for i := 0; i < days; i++ {
		d := from.AddDate(0, 0, i)
		m, ok := Matches(kind, rule, d)
		if !ok {
			return time.Time{}, false, false
		}
		if m {
			return d, true, true
		}
	}
	return time.Time{}, false, true
}

// Matches reports whether the rule of kind matches day; ok false when it cannot be evaluated.
func Matches(kind string, rule *pyjson.Object, day time.Time) (match, ok bool) {
	switch kind {
	case "date":
		return dateMatches(rule, day)
	case "date_range":
		return rangeMatches(rule, day)
	case "week_and_day":
		return all(day, monthOf(field(rule, "month")), weekOf(field(rule, "week")), weekdayOf(field(rule, "weekday")))
	case "custom":
		return all(day, yearOf(field(rule, "year")), monthOf(listOrAny(field(rule, "months"))),
			dayOf(listOrAny(field(rule, "days"))), weekOf(listOrAny(field(rule, "weeks"))),
			weekdayOf(listOrAny(field(rule, "weekdays"))))
	}
	return false, false
}

func dateMatches(rule *pyjson.Object, day time.Time) (bool, bool) {
	return all(day, yearOf(field(rule, "year")), monthOf(field(rule, "month")), dayOf(field(rule, "day")),
		weekdayOf(field(rule, "weekday")))
}

// matcher is one rule field: whether it matches a day, ok false when it cannot be evaluated.
type matcher func(time.Time) (bool, bool)

func all(day time.Time, ms ...matcher) (bool, bool) {
	match := true
	for _, m := range ms {
		hit, ok := m(day)
		if !ok {
			return false, false
		}
		match = match && hit
	}
	return match, true
}

// listOrAny is a custom rule's list, "*" when empty (an empty list matches every day).
func listOrAny(v any) any {
	if l, ok := v.([]any); ok && len(l) == 0 {
		return "*"
	}
	return v
}

// anyOf matches when v is "*", or one value (or one entry of a list) matches by one.
func anyOf(v any, one func(any, time.Time) (bool, bool)) matcher {
	return func(day time.Time) (bool, bool) {
		if v == "*" {
			return true, true
		}
		list, isList := v.([]any)
		if !isList {
			list = []any{v}
		}
		hit := false
		for _, x := range list {
			m, ok := one(x, day)
			if !ok {
				return false, false
			}
			hit = hit || m
		}
		return hit, true
	}
}

func yearOf(v any) matcher {
	return anyOf(v, func(x any, d time.Time) (bool, bool) { y, ok := x.(int); return y == d.Year(), ok })
}

func dayOf(v any) matcher {
	return anyOf(v, func(x any, d time.Time) (bool, bool) {
		if x == "*" {
			return true, true
		}
		n, ok := x.(int)
		return n == d.Day(), ok
	})
}

func monthOf(v any) matcher {
	return anyOf(v, func(x any, d time.Time) (bool, bool) {
		m := int(d.Month())
		switch x {
		case "odd":
			return m%2 == 1, true
		case "even":
			return m%2 == 0, true
		}
		s, _ := x.(string)
		i := enumOrdinal(Months[:12], s)
		return i+1 == m, i >= 0
	})
}

func weekdayOf(v any) matcher {
	return anyOf(v, func(x any, d time.Time) (bool, bool) {
		s, _ := x.(string)
		i := enumOrdinal(Weekdays, s)
		return i == int(d.Weekday()), i >= 0
	})
}

// weekOf evaluates week1..week5 (days 1-7, 8-14, ...) and last7days; the calendar weeks and the
// weeks before the last are not evaluated.
func weekOf(v any) matcher {
	return anyOf(v, func(x any, d time.Time) (bool, bool) {
		s, _ := x.(string)
		switch s {
		case "*":
			return true, true
		case "week1", "week2", "week3", "week4", "week5":
			return (d.Day()-1)/7+1 == int(s[4]-'0'), true
		case "last7days":
			last := time.Date(d.Year(), d.Month()+1, 0, 0, 0, 0, 0, d.Location()).Day()
			return d.Day() > last-7, true
		}
		return false, false
	})
}

// rangeMatches evaluates a date range whose ends are whole dates, or whose ends name a month
// and day in every year (a yearly range, which may wrap past december); any other wildcard, or
// a weekday on an end, is not evaluated.
func rangeMatches(rule *pyjson.Object, day time.Time) (bool, bool) {
	start, _ := field(rule, "start").(*pyjson.Object)
	end, _ := field(rule, "end").(*pyjson.Object)
	sy, sm, sd, ok1 := wholeDate(start)
	ey, em, ed, ok2 := wholeDate(end)
	if !ok1 || !ok2 || (sy == 0) != (ey == 0) {
		return false, false
	}
	key := func(y, m, d int) int { return y*10000 + m*100 + d }
	if sy != 0 {
		k := key(day.Year(), int(day.Month()), day.Day())
		return k >= key(sy, sm, sd) && k <= key(ey, em, ed), true
	}
	k, from, to := key(0, int(day.Month()), day.Day()), key(0, sm, sd), key(0, em, ed)
	if from <= to {
		return k >= from && k <= to, true
	}
	return k >= from || k <= to, true
}

// wholeDate is a range end's year (0 for every year), month and day; ok false for any other
// wildcard, a month alternation or a weekday.
func wholeDate(d *pyjson.Object) (y, m, day int, ok bool) {
	if d == nil || field(d, "weekday") != "*" && field(d, "weekday") != nil {
		return 0, 0, 0, false
	}
	switch v := field(d, "year").(type) {
	case int:
		y = v
	case string:
		if v != "*" {
			return 0, 0, 0, false
		}
	default:
		return 0, 0, 0, false
	}
	s, _ := field(d, "month").(string)
	m = enumOrdinal(Months[:12], s) + 1
	day, isInt := field(d, "day").(int)
	return y, m, day, m >= 1 && isInt
}
