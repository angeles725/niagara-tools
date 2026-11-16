package schedule

import (
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// ranges are the time ranges under the DaySchedule at path, sorted by start: {start, end,
// value[, status]}, end "24:00" for a range that runs to midnight. A range without a value is
// returned with value null and ignored true: the station skips it.
func (t *tree) ranges(path string, k *Kind) ([]any, error) {
	type item struct {
		start string
		o     *pyjson.Object
	}
	var items []item
	for _, p := range t.schedules(path) {
		if t.typeOf(p) != "schedule:TimeSchedule" {
			continue
		}
		start, err := t.simple(p, "start")
		if err != nil {
			return nil, err
		}
		finish, err := t.simple(p, "finish")
		if err != nil {
			return nil, err
		}
		end := clock(finish)
		if end == "00:00" {
			end = "24:00"
		}
		o := obj("start", clock(start), "end", end)
		if k.ValueType != "" {
			if err := t.rangeValue(p, o); err != nil {
				return nil, err
			}
		}
		items = append(items, item{clock(start), o})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].start < items[j].start })
	out := []any{}
	for _, it := range items {
		out = append(out, it.o)
	}
	return out, nil
}

// rangeValue adds value (and status when not ok) of the range at p, or ignored when it has none.
func (t *tree) rangeValue(p string, o *pyjson.Object) error {
	if t.node(join(p, "effectiveValue")) == nil {
		o.Set("value", nil)
		o.Set("ignored", true)
		return nil
	}
	v, err := t.status(p, "effectiveValue")
	if err != nil {
		return err
	}
	value, _ := v.Get("value")
	o.Set("value", value)
	if s, _ := v.Get("status"); s != "ok" {
		o.Set("status", s)
	}
	return nil
}

// clock is an encoded BTime ("08:00:00.000") as "08:00", keeping seconds and milliseconds only
// when they are not zero.
func clock(v string) string {
	v = strings.TrimSuffix(v, ".000")
	return strings.TrimSuffix(v, ":00")
}

// dates adds a calendar's dates (its own children) or a trigger's (those of its dates calendar).
func (t *tree) dates(k *Kind, out *pyjson.Object) error {
	if !k.Dates {
		return nil
	}
	base := ""
	if k.Times {
		base = "dates"
	}
	list := []any{}
	for _, p := range t.schedules(base) {
		if eventKindOf(t.typeOf(p)) == "" {
			continue
		}
		e, err := t.event(p, p)
		if err != nil {
			return err
		}
		list = append(list, e)
	}
	out.Set("dates", list)
	return nil
}

// trigger adds enabled, the firing times (the start of each times range), and the last and next
// trigger.
func (t *tree) trigger(k *Kind, out *pyjson.Object) error {
	if !k.Times {
		return nil
	}
	enabled, err := t.simple("", "enabled")
	if err != nil {
		return err
	}
	out.Set("enabled", enabled == "true")
	times := []any{}
	ranges, err := t.ranges("times", k)
	if err != nil {
		return err
	}
	for _, r := range ranges {
		start, _ := r.(*pyjson.Object).Get("start")
		times = append(times, start)
	}
	out.Set("times", times)
	for _, s := range [][2]string{{"last_trigger", "lastTrigger"}, {"next_trigger", "nextTrigger"}} {
		v, err := t.simple("", s[1])
		if err != nil {
			return err
		}
		out.Set(s[0], absTime(v))
	}
	return nil
}
