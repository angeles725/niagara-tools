package schedule

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// EncodeRule is the BOX body of a day rule given in n4_read_schedule's shape (the rule of a
// special event or a calendar date), each fault added to f under where. The frozen children
// carry singleSelection true where the type registry declares it (DateSchedule and
// WeekAndDaySchedule children, live 4.15.6.20), as Workbench creates them.
func EncodeRule(kind string, rule *pyjson.Object, where string, f *Faults) *pyjson.Object {
	r := ruleReader{rule, where, f}
	switch kind {
	case "date":
		r.only("year", "month", "day", "weekday")
		return r.date(rule, where, "")
	case "date_range":
		r.only("start", "end")
		return obj("nm", "p", "t", "schedule:DateRangeSchedule", "s", []any{
			r.date(r.object("start"), where+".start", "start"), r.date(r.object("end"), where+".end", "end")})
	case "week_and_day":
		r.only("month", "week", "weekday")
		return obj("nm", "p", "t", "schedule:WeekAndDaySchedule", "s", []any{
			r.set("months", "schedule:MonthSchedule", "month", true), r.set("weeksOfMonth", "schedule:WeekOfMonthSchedule", "week", true),
			r.set("weekdays", "schedule:WeekdaySchedule", "weekday", true)})
	case "custom":
		r.only("year", "months", "days", "weeks", "weekdays")
		return obj("nm", "p", "t", "schedule:CustomSchedule", "s", []any{
			r.set("daysOfMonth", "schedule:DayOfMonthSchedule", "days", false), r.set("months", "schedule:MonthSchedule", "months", false),
			r.set("weekdays", "schedule:WeekdaySchedule", "weekdays", false),
			r.set("weeksOfMonth", "schedule:WeekOfMonthSchedule", "weeks", false), r.year("year", "year")})
	case "reference":
		r.only("calendar")
		cal, _ := field(rule, "calendar").(string)
		if !strings.HasPrefix(cal, "station:|slot:/") {
			f.add("%s.calendar: %s is not a calendar's slot ORD (station:|slot:/...)", where, pyjson.Repr(field(rule, "calendar")))
		}
		return obj("nm", "p", "t", "schedule:ScheduleReference", "s", []any{
			obj("nm", "p", "n", "ref", "t", "baja:Ord", "v", cal)})
	}
	f.add("%s: unknown kind %s (use date, date_range, week_and_day, custom or reference)", where, pyjson.Repr(kind))
	return nil
}

// ruleReader reads the fields of one rule object, adding each fault to f under where.
type ruleReader struct {
	rule  *pyjson.Object
	where string
	f     *Faults
}

func (r ruleReader) only(keys ...string) {
	if r.rule == nil {
		return
	}
	for _, k := range r.rule.Keys() {
		if !known(keys, k) {
			r.f.add("%s: unknown field %s (use %s)", r.where, pyjson.StrRepr(k), strings.Join(keys, ", "))
		}
	}
}

func (r ruleReader) object(k string) *pyjson.Object {
	o, ok := field(r.rule, k).(*pyjson.Object)
	if !ok {
		r.f.add("%s.%s: must be a date object {year, month, day, weekday}", r.where, k)
	}
	return o
}

// date is a DateSchedule body named name ("" for an event's own days).
func (r ruleReader) date(rule *pyjson.Object, where, name string) *pyjson.Object {
	d := ruleReader{rule, where, r.f}
	if name != "" {
		d.only("year", "month", "day", "weekday")
	}
	body := obj("nm", "p", "t", "schedule:DateSchedule", "s", []any{d.year("yearSchedule", "year"),
		d.set("monthSchedule", "schedule:MonthSchedule", "month", true), d.set("daySchedule", "schedule:DayOfMonthSchedule", "day", true),
		d.set("weekdaySchedule", "schedule:WeekdaySchedule", "weekday", true)})
	if name != "" {
		body.Set("n", name)
	}
	return body
}

// year is a YearSchedule child: always effective for "*" (or absent), else the year. The type's
// own alwaysEffective default is false, so "*" is always sent.
func (r ruleReader) year(slot, k string) *pyjson.Object {
	v := field(r.rule, k)
	kids := []any{obj("nm", "p", "n", "alwaysEffective", "t", "baja:Boolean", "v", "true")}
	if y, ok := intOf(v); ok && y >= 1970 && y <= 9999 {
		kids = []any{obj("nm", "p", "n", "alwaysEffective", "t", "baja:Boolean", "v", "false"),
			obj("nm", "p", "n", "year", "t", "baja:Integer", "v", strconv.Itoa(y))}
	} else if v != nil && v != "*" {
		r.f.add("%s.%s: %s is not a year (1970-9999) or \"*\"", r.where, k, pyjson.Repr(v))
	}
	return obj("nm", "p", "n", slot, "t", "schedule:YearSchedule", "s", kids)
}

// set is an EnumSet child of type typ: the ordinals of field k ("*" or absent: empty, any).
func (r ruleReader) set(slot, typ, k string, single bool) *pyjson.Object {
	kids := []any{}
	if single {
		kids = append(kids, obj("nm", "p", "n", "singleSelection", "t", "baja:Boolean", "v", "true"))
	}
	if ords := r.ordinals(k); len(ords) > 0 {
		kids = append(kids, obj("nm", "p", "n", "set", "t", "baja:EnumSet", "v", strings.Join(ords, ",")))
	}
	return obj("nm", "p", "n", slot, "t", typ, "s", kids)
}

// ordinals reads field k (a name, a day number, "*", or a list of them) as set ordinals.
func (r ruleReader) ordinals(k string) []string {
	v := field(r.rule, k)
	items, isList := v.([]any)
	if !isList {
		if v == nil || v == "*" {
			return nil
		}
		items = []any{v}
	}
	var out []string
	for _, item := range items {
		if n := ordinalOf(k, item); n >= 0 {
			out = append(out, strconv.Itoa(n))
		} else {
			r.f.add("%s.%s: %s is not %s", r.where, k, pyjson.Repr(item), tableHelp(k))
		}
	}
	return out
}

// ordinalOf is the set ordinal of one value of field k, -1 when it is none.
func ordinalOf(k string, v any) int {
	s, _ := v.(string)
	switch strings.TrimSuffix(k, "s") {
	case "month":
		return enumOrdinal(Months, s)
	case "weekday":
		return enumOrdinal(Weekdays, s)
	case "week":
		if n := enumOrdinal(Weeks, s); n > 0 {
			return n
		}
		return -1
	}
	if n, ok := intOf(v); ok && n >= 1 && n <= MaxDay {
		return n
	}
	return -1
}

func tableHelp(k string) string {
	switch strings.TrimSuffix(k, "s") {
	case "month":
		return "a month (" + strings.Join(Months, ", ") + ")"
	case "weekday":
		return "a weekday (" + strings.Join(Weekdays, ", ") + ")"
	case "week":
		return "a week (" + strings.Join(Weeks[1:], ", ") + ")"
	}
	return fmt.Sprintf("a day of the month (1-%d)", MaxDay)
}
