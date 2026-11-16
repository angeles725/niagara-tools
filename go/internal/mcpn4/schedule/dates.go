package schedule

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// DateKinds are the rule kinds a calendar date may have (a calendar cannot reference itself).
var DateKinds = []string{"date", "date_range", "week_and_day", "custom"}

// Date is one requested calendar date: its name (shown and as a slot name), kind, rule and the
// rule's BOX body.
type Date struct {
	Name, Slot, Kind string
	Rule, Body       *pyjson.Object
}

// Dates parses add_date: [{name, kind, rule}]. As for special events, a date or date range that
// ends before today is a fault unless allowExpired.
func Dates(list []any, today time.Time, allowExpired bool, f *Faults) []Date {
	var out []Date
	for i, item := range list {
		where := fmt.Sprintf("add_date[%d]", i)
		o, ok := item.(*pyjson.Object)
		if !ok {
			f.add("%s: must be an object {name, kind, rule}", where)
			continue
		}
		ruleReader{o, where, f}.only("name", "kind", "rule")
		name, _ := field(o, "name").(string)
		if strings.TrimSpace(name) == "" {
			f.add("%s.name: must be a non-empty string", where)
		}
		kind, _ := field(o, "kind").(string)
		rule, _ := field(o, "rule").(*pyjson.Object)
		if rule == nil {
			rule = pyjson.NewObject()
		}
		if !known(DateKinds, kind) {
			f.add("%s: kind %s is not a calendar date kind (use %s)", where, pyjson.Repr(field(o, "kind")), strings.Join(DateKinds, ", "))
			continue
		}
		body := EncodeRule(kind, rule, where+".rule", f)
		if Expired(kind, rule, today) && !allowExpired {
			f.add("%s: %s can never occur again (it ends before %s); pass allow_expired: true to add it anyway",
				where, pyjson.StrRepr(name), today.Format("2006-01-02"))
		}
		out = append(out, Date{name, EscapeName(name), kind, rule, body})
	}
	return out
}

// RuleView is a requested rule as n4_read_schedule shows it: every field present ("*" for any),
// a one-item list as its item, custom fields always lists.
func RuleView(kind string, rule *pyjson.Object) *pyjson.Object {
	switch kind {
	case "date":
		return viewFields(rule, []string{"year", "month", "day", "weekday"}, false)
	case "date_range":
		start, _ := field(rule, "start").(*pyjson.Object)
		end, _ := field(rule, "end").(*pyjson.Object)
		return obj("start", RuleView("date", orEmpty(start)), "end", RuleView("date", orEmpty(end)))
	case "week_and_day":
		return viewFields(rule, []string{"month", "week", "weekday"}, false)
	case "custom":
		out := viewFields(rule, []string{"year"}, false)
		lists := viewFields(rule, []string{"months", "days", "weeks", "weekdays"}, true)
		for _, k := range lists.Keys() {
			out.Set(k, field(lists, k))
		}
		return out
	}
	return obj("calendar", field(rule, "calendar"))
}

func viewFields(rule *pyjson.Object, keys []string, lists bool) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range keys {
		v := field(rule, k)
		items, isList := v.([]any)
		switch {
		case lists && (v == nil || v == "*"):
			out.Set(k, []any{})
		case lists && !isList:
			out.Set(k, []any{v})
		case !lists && v == nil, !lists && isList && len(items) == 0:
			out.Set(k, "*")
		case !lists && isList && len(items) == 1:
			out.Set(k, items[0])
		default:
			out.Set(k, v)
		}
	}
	return out
}

func orEmpty(o *pyjson.Object) *pyjson.Object {
	if o == nil {
		return pyjson.NewObject()
	}
	return o
}

// TriggerTimes parses times: a list of HH:MM, sorted; a malformed or repeated time is a fault.
func TriggerTimes(list []any, f *Faults) []int {
	seen := map[int]bool{}
	var out []int
	for i, v := range list {
		m := minutes(v, false)
		switch {
		case m < 0:
			f.add("times[%d]: %s is not a time HH:MM from 00:00 to 23:59", i, pyjson.Repr(v))
		case seen[m]:
			f.add("times[%d]: %s is given twice", i, pyjson.Repr(v))
		default:
			seen[m] = true
			out = append(out, m)
		}
	}
	sort.Ints(out)
	return out
}

// TimesBody is a trigger's times as the station stores them: a DaySchedule of TimeSchedules
// without a value, each 20 seconds long (live 4.15.6.20 trigger times).
func TimesBody(times []int) *pyjson.Object {
	kids := []any{}
	for i, m := range times {
		name := "time"
		if i > 0 {
			name = fmt.Sprintf("time%d", i)
		}
		kids = append(kids, obj("nm", "p", "n", name, "t", "schedule:TimeSchedule", "s", []any{
			obj("nm", "p", "n", "start", "t", "baja:Time", "v", btime(m)),
			obj("nm", "p", "n", "finish", "t", "baja:Time", "v", fmt.Sprintf("%02d:%02d:20.000", m/60, m%60))}))
	}
	return obj("nm", "p", "t", "schedule:DaySchedule", "s", kids)
}
