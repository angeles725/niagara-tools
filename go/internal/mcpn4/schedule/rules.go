package schedule

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// eventKindOf is the special-event kind of a day-rule type, "" for any other type.
func eventKindOf(typ string) string {
	for _, e := range EventKinds {
		if e.Type == typ {
			return e.Name
		}
	}
	return ""
}

// rule decodes the day rule at path (one of the EventKinds types) into its plain fields.
func (t *tree) rule(path string) (*pyjson.Object, error) {
	switch eventKindOf(t.typeOf(path)) {
	case "date":
		return t.date(path)
	case "date_range":
		start, err := t.date(join(path, "start"))
		if err != nil {
			return nil, err
		}
		end, err := t.date(join(path, "end"))
		return obj("start", start, "end", end), err
	case "week_and_day":
		return t.fields(path, [][3]string{{"month", "months", "months"}, {"week", "weeksOfMonth", "weeks"},
			{"weekday", "weekdays", "weekdays"}}, false)
	case "custom":
		year, err := t.year(join(path, "year"))
		if err != nil {
			return nil, err
		}
		rest, err := t.fields(path, [][3]string{{"months", "months", "months"}, {"days", "daysOfMonth", "days"},
			{"weeks", "weeksOfMonth", "weeks"}, {"weekdays", "weekdays", "weekdays"}}, true)
		if err != nil {
			return nil, err
		}
		out := obj("year", year)
		for _, k := range rest.Keys() {
			v, _ := rest.Get(k)
			out.Set(k, v)
		}
		return out, nil
	case "reference":
		ref, err := t.simple(path, "ref")
		var calendar any
		if ref != "null" && ref != "" {
			calendar = ref
		}
		return obj("calendar", calendar), err
	}
	return nil, nil
}

// date decodes a DateSchedule: year, month, day and weekday, each "*" when it matches any.
func (t *tree) date(path string) (*pyjson.Object, error) {
	year, err := t.year(join(path, "yearSchedule"))
	if err != nil {
		return nil, err
	}
	rest, err := t.fields(path, [][3]string{{"month", "monthSchedule", "months"}, {"day", "daySchedule", "days"},
		{"weekday", "weekdaySchedule", "weekdays"}}, false)
	if err != nil {
		return nil, err
	}
	out := obj("year", year)
	for _, k := range rest.Keys() {
		v, _ := rest.Get(k)
		out.Set(k, v)
	}
	return out, nil
}

// year is a YearSchedule: "*" when always effective, else the year.
func (t *tree) year(path string) (any, error) {
	always, err := t.simple(path, "alwaysEffective")
	if err != nil || always == "true" {
		return "*", err
	}
	y, err := t.simple(path, "year")
	if err != nil {
		return nil, err
	}
	n, convErr := strconv.Atoi(y)
	if convErr != nil {
		return y, nil
	}
	return n, nil
}

// fields decodes the enum-set children of path: each spec is {output name, slot, table}. An
// empty set that is effective when empty is "*"; a list field (custom) is always a list.
func (t *tree) fields(path string, specs [][3]string, lists bool) (*pyjson.Object, error) {
	out := pyjson.NewObject()
	for _, s := range specs {
		p := join(path, s[1])
		set, err := t.simple(p, "set")
		if err != nil {
			return nil, err
		}
		set, _, _ = strings.Cut(set, "@")
		whenEmpty, err := t.simple(p, "effectiveWhenEmpty")
		if err != nil {
			return nil, err
		}
		values := []any{}
		for _, part := range strings.Split(set, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
				values = append(values, ordinalName(s[2], n))
			}
		}
		switch {
		case lists:
			out.Set(s[0], values)
		case len(values) == 0 && whenEmpty == "true", len(values) == 1 && values[0] == "*":
			out.Set(s[0], "*")
		case len(values) == 1:
			out.Set(s[0], values[0])
		default:
			out.Set(s[0], values)
		}
	}
	return out, nil
}

// ordinalName names ordinal n of a set table ("months", "weekdays", "weeks", "days").
func ordinalName(table string, n int) any {
	switch table {
	case "months":
		return enumName(Months, n)
	case "weekdays":
		return enumName(Weekdays, n)
	case "weeks":
		return enumName(Weeks, n)
	}
	if n == 0 {
		return "*"
	}
	return n
}
