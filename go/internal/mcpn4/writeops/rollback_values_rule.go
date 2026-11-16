package writeops

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ruleTypes are the day-rule types whose content the rollback compares: what a special event's
// days and a calendar date hold, and the parts a date is set by path.
var ruleTypes = map[string]bool{"schedule:DateSchedule": true, "schedule:DateRangeSchedule": true,
	"schedule:WeekAndDaySchedule": true, "schedule:CustomSchedule": true, "schedule:ScheduleReference": true,
	"schedule:YearSchedule": true, "schedule:MonthSchedule": true, "schedule:DayOfMonthSchedule": true,
	"schedule:WeekdaySchedule": true, "schedule:WeekOfMonthSchedule": true}

func ruleType(t string) bool { return ruleTypes[t] }

// ruleDefaults are the defaults the encoder sends and a station leaves out of a rule (a year's
// alwaysEffective, an empty set): such a leaf counts as absent, so a body as sent and the rule as
// loaded give the same text.
var ruleDefaults = map[string]string{"alwaysEffective": "false", "set": ""}

// ruleText is a rule subtree as one comparable text: its leaves "path=value", sorted, without
// the defaults, the selection mode (singleSelection, an editor setting) and the year of an
// always-effective YearSchedule; an enum set's @range suffix is cut.
func ruleText(n *pyjson.Object) string {
	var out []string
	var walk func(n *pyjson.Object, path string)
	walk = func(n *pyjson.Object, path string) {
		kids, _ := sOf(n).([]any)
		always := childV(n, "alwaysEffective", "false") == "true"
		for _, k := range kids {
			c, ok := k.(*pyjson.Object)
			if !ok {
				continue
			}
			name := pyjson.Str(get(c, "n"))
			if _, nested := get(c, "s").([]any); nested || box.IsComponentType(get(c, "t")) { // an empty part loads without s
				walk(c, path+name+"/")
				continue
			}
			v := pyjson.Str(get(c, "v"))
			v, _, _ = strings.Cut(v, "@")
			if def, known := ruleDefaults[name]; name == "singleSelection" || known && v == def || name == "year" && always {
				continue
			}
			out = append(out, path+name+"="+v)
		}
	}
	walk(n, "")
	sort.Strings(out)
	return strings.Join(out, "; ")
}

// eventsText is the specialEvents list as one comparable text: each event in order (the order is
// the priority) with its name, ranges and rule.
func eventsText(list *pyjson.Object) string {
	var out []string
	kids, _ := sOf(list).([]any)
	for _, k := range kids {
		e, ok := k.(*pyjson.Object)
		if !ok {
			continue
		}
		day, days := child(e, "day"), child(e, "days")
		text := pyjson.Str(get(e, "n")) + " {" + dayText(orNew(day)) + "} {" + pyjson.Str(get(days, "t")) + " " + ruleText(orNew(days)) + "}"
		out = append(out, text)
	}
	return strings.Join(out, " | ")
}

func orNew(o *pyjson.Object) *pyjson.Object {
	if o == nil {
		return pyjson.NewObject()
	}
	return o
}
