package schedule

import (
	"fmt"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// LoadDepth is the load depth that reaches every slot Read uses (a date range special event:
// schedule/specialEvents/<event>/days/start/yearSchedule/year, plus value children).
const LoadDepth = 9

// KindOf is the kind whose station type is typ, nil when typ is no schedule kind.
func KindOf(typ string) *Kind {
	for i := range Kinds {
		if Kinds[i].Type == typ {
			return &Kinds[i]
		}
	}
	return nil
}

// Read decodes a schedule component loaded to LoadDepth ({path: node}, "" the schedule) into its
// plain model: kind, out and what drives it, then the weekly ranges and special events, the
// calendar dates or the trigger dates and times, by kind.
func Read(nodes *pyjson.Object, def Defaults) (*pyjson.Object, error) {
	t := &tree{nodes: nodes, def: def}
	k := KindOf(t.typeOf(""))
	if k == nil {
		return nil, fmt.Errorf("%s is not a schedule kind; n4_list_schedule_kinds lists them", t.typeOf(""))
	}
	out := obj("kind", k.Name, "type", k.Type)
	steps := []func(*Kind, *pyjson.Object) error{t.outputs, t.weekly, t.events, t.dates, t.trigger}
	for _, step := range steps {
		if err := step(k, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// outputs adds out, outSource, next time and value, default output, facets and lastModified.
func (t *tree) outputs(k *Kind, out *pyjson.Object) error {
	if k.ValueType != "" {
		for _, s := range [][2]string{{"out", "out"}, {"next_value", "nextValue"}, {"default_output", "defaultOutput"},
			{"in", "in"}} {
			v, err := t.status("", s[1])
			if err != nil {
				return err
			}
			out.Set(s[0], v)
		}
		next, err := t.simple("", "nextTime")
		if err != nil {
			return err
		}
		out.Set("next_time", absTime(next))
		facets, err := t.simple("", "facets")
		if err != nil {
			return err
		}
		out.Set("facets", facets)
	}
	if k.Weekly {
		src, err := t.simple("", "outSource")
		if err != nil {
			return err
		}
		out.Set("out_source", src)
	}
	modified, err := t.simple("", "lastModified")
	out.Set("last_modified", absTime(modified))
	return err
}

// weekly adds the effective period and the ranges of each day, monday first.
func (t *tree) weekly(k *Kind, out *pyjson.Object) error {
	if !k.Weekly {
		return nil
	}
	effective, err := t.rule("effective")
	if err != nil {
		return err
	}
	if always(effective) {
		out.Set("effective", "always")
	} else {
		out.Set("effective", effective)
	}
	week := pyjson.NewObject()
	for _, d := range Days {
		ranges, err := t.ranges(join("schedule/week/"+d, "day"), k)
		if err != nil {
			return err
		}
		week.Set(d, ranges)
	}
	out.Set("weekly", week)
	return nil
}

// events adds the special events, highest priority first.
func (t *tree) events(k *Kind, out *pyjson.Object) error {
	if !k.SpecialEvents {
		return nil
	}
	list := []any{}
	for i, p := range t.schedules("schedule/specialEvents") {
		e, err := t.event(join(p, "days"), p)
		if err != nil {
			return err
		}
		ranges, err := t.ranges(join(p, "day"), k)
		if err != nil {
			return err
		}
		e.Set("priority", i+1)
		e.Set("ranges", ranges)
		list = append(list, e)
	}
	out.Set("special_events", list)
	return nil
}

// event is {name, kind, rule} of the day rule at rulePath, named after the slot at named.
func (t *tree) event(rulePath, named string) (*pyjson.Object, error) {
	rule, err := t.rule(rulePath)
	if err != nil {
		return nil, err
	}
	kind := eventKindOf(t.typeOf(rulePath))
	return obj("name", bql.UnescapeSlot(lastName(named)), "kind", kind, "rule", rule), nil
}

// always is true when a date range rule matches every day (both ends "*" everywhere).
func always(r *pyjson.Object) bool {
	for _, end := range []string{"start", "end"} {
		d, _ := field(r, end).(*pyjson.Object)
		for _, k := range d.Keys() {
			if v, _ := d.Get(k); v != "*" {
				return false
			}
		}
	}
	return true
}

func lastName(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

// absTime is an encoded BAbsTime as given, nil for the null time (BAbsTime.NULL is the epoch,
// written in the station's time zone).
func absTime(v string) any {
	at, err := time.Parse("2006-01-02T15:04:05.000Z07:00", v)
	if v == "" || v == "null" || err == nil && at.UnixMilli() == 0 {
		return nil
	}
	return v
}
