package writeops

import (
	"sort"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The rollback compares most set slots by their plain value (plan.Observe). Two kinds are
// compared by what they hold instead: a StatusEnum or StatusString by its text and status
// (rollback_values_status.go) and a schedule DaySchedule (written whole by n4_set_schedule) by
// its time ranges. Every other body keeps the generic comparison, unchanged.

const daySchedule = "schedule:DaySchedule"

// dayDepth is how far below a day its ranges' values sit (day/time/effectiveValue/value).
const dayDepth = 4

// opDepth is the load depth that observes the slot a set op writes: slotDepth, and a day's
// ranges too.
func opDepth(op *pyjson.Object, slot string) int {
	if b, ok := get(op, "b").(*pyjson.Object); ok && get(b, "t") == daySchedule {
		return slotDepth(slot) + dayDepth
	}
	return slotDepth(slot)
}

// ownValue is the value of a body of an own-compared type (ok false for any other).
func ownValue(t string, b *pyjson.Object) (any, bool) {
	switch {
	case t == daySchedule:
		return dayText(b), true
	case textStatus(t):
		kids, _ := sOf(b).([]any)
		values := map[string]any{}
		for _, k := range kids {
			if c, ok := k.(*pyjson.Object); ok {
				values[pyjson.Str(get(c, "n"))] = get(c, "v")
			}
		}
		return textStatusValue(t, values["value"], values["status"]), true
	}
	return nil, false
}

// observeOwn is the loaded slot of an own-compared type t (nil when it is gone or of another
// type: it changed); ok false for any other type.
func observeOwn(nodes *pyjson.Object, slot, t string) (any, bool) {
	switch {
	case t == daySchedule:
		n := box.Node(nodes, slot)
		if n == nil || get(n, "t") != t {
			return nil, true
		}
		return dayText(n), true
	case textStatus(t):
		return observeTextStatus(nodes, slot, t), true
	}
	return nil, false
}

// dayText is a day's ranges, sorted, as one comparable text: "start-finish value/status" per
// range, "ignored" for a range without an effectiveValue. Children the station leaves out take
// their defaults, so a body as sent and the same day as loaded give the same text.
func dayText(day *pyjson.Object) string {
	var out []string
	kids, _ := sOf(day).([]any)
	for _, k := range kids {
		r, ok := k.(*pyjson.Object)
		if !ok || get(r, "t") != "schedule:TimeSchedule" {
			continue
		}
		start, finish := childV(r, "start", "00:00:00.000"), childV(r, "finish", "00:00:00.000")
		value := "ignored"
		if ev := child(r, "effectiveValue"); ev != nil {
			t := pyjson.Str(get(ev, "t"))
			v := childV(ev, "value", map[string]string{"baja:StatusBoolean": "false", "baja:StatusNumeric": "0.0"}[t]+textDefault[t])
			if t == "baja:StatusEnum" {
				v, _, _ = strings.Cut(v, "@")
			} else if f, err := strconv.ParseFloat(v, 64); err == nil && t == "baja:StatusNumeric" {
				v = strconv.FormatFloat(f, 'g', -1, 64)
			}
			value = t + " " + v + "/" + childV(ev, "status", "0")
		}
		out = append(out, start+"-"+finish+" "+value)
	}
	sort.Strings(out)
	return strings.Join(out, "; ")
}

func child(n *pyjson.Object, name string) *pyjson.Object {
	kids, _ := sOf(n).([]any)
	for _, k := range kids {
		if c, ok := k.(*pyjson.Object); ok && get(c, "n") == name {
			return c
		}
	}
	return nil
}

func childV(n *pyjson.Object, name, def string) string {
	if c := child(n, name); c != nil {
		if v, ok := get(c, "v").(string); ok && v != "" {
			return v
		}
	}
	return def
}
