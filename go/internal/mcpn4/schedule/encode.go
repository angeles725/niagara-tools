package schedule

import (
	"fmt"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The BOX bodies a write sends: a day of ranges, a range, a Status value. Range values are sent
// as the station's own loads show them (live 4.15.6.20: effectiveValue with flags o1, the value
// child plain, the status "0").

// encodedValue is a checked plain value as the station encodes it for kind k.
func encodedValue(k *Kind, v any) string {
	switch k.Name {
	case "boolean":
		return strconv.FormatBool(v.(bool))
	case "numeric":
		f, _ := floatOf(v)
		return pyjson.FloatNumber(f).String()
	case "enum":
		n, _ := intOf(v)
		return strconv.Itoa(n)
	}
	return v.(string)
}

// StatusBody is the BOX body of a Status value of kind k: value v with status ok, or the type
// default with status null when v is nil.
func StatusBody(k *Kind, v any) *pyjson.Object {
	value, status := "", "0"
	if v == nil {
		status = "40"
		value = map[string]string{"boolean": "false", "numeric": "0.0", "enum": "0"}[k.Name]
	} else {
		value = encodedValue(k, v)
	}
	kid := obj("nm", "p", "n", "value", "v", value)
	if t := map[string]string{"boolean": "baja:Boolean", "numeric": "baja:Double", "string": "baja:String"}[k.Name]; t != "" {
		kid = obj("nm", "p", "n", "value", "t", t, "v", value)
	}
	return obj("nm", "p", "t", k.ValueType, "s", []any{kid, obj("nm", "p", "n", "status", "t", "baja:Status", "v", status)})
}

// DayBody is the BOX body of a DaySchedule holding ranges, named time, time1, ... as the
// station names the ranges it adds.
func DayBody(k *Kind, ranges []Range) *pyjson.Object {
	kids := []any{}
	for i, r := range ranges {
		name := "time"
		if i > 0 {
			name = fmt.Sprintf("time%d", i)
		}
		ev := StatusBody(k, r.Value)
		ev.Set("n", "effectiveValue")
		ev.Set("f", "o1")
		kids = append(kids, obj("nm", "p", "n", name, "t", "schedule:TimeSchedule", "s", []any{
			obj("nm", "p", "n", "start", "t", "baja:Time", "v", btime(r.Start)),
			obj("nm", "p", "n", "finish", "t", "baja:Time", "v", btime(r.End)), ev}))
	}
	return obj("nm", "p", "t", "schedule:DaySchedule", "s", kids)
}

// btime is minutes of the day as an encoded BTime; 1440 (midnight at the end) is 00:00.
func btime(m int) string { return fmt.Sprintf("%02d:%02d:00.000", m/60%24, m%60) }

// View is a requested range as n4_read_schedule shows it, for the read-back comparison.
func View(k *Kind, r Range) *pyjson.Object {
	return obj("start", hhmm(r.Start), "end", hhmm(r.End), "value", plain(k.ValueType, encodedValue(k, r.Value)))
}

// Strip is a loaded node as the body of a set that puts it back: only nm, t, v, f and s, and n
// below the top, recursively (handles and display strings dropped), so the set restores exactly
// what was loaded (the station leaves out the same defaults it will fill in).
func Strip(n *pyjson.Object) *pyjson.Object { return strip(n, false) }

func strip(n *pyjson.Object, named bool) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range []string{"nm", "n", "t", "v", "f"} {
		if v, ok := n.Get(k); ok && (named || k != "n") {
			out.Set(k, v)
		}
	}
	if subs, ok := field(n, "s").([]any); ok {
		kids := make([]any, 0, len(subs))
		for _, s := range subs {
			if o, ok := s.(*pyjson.Object); ok {
				kids = append(kids, strip(o, true))
			}
		}
		out.Set("s", kids)
	}
	return out
}

// StatusView is a requested Status value as n4_read_schedule shows it ({value, status}).
func StatusView(k *Kind, v any) *pyjson.Object {
	if v == nil {
		return obj("value", nil, "status", "null")
	}
	return obj("value", plain(k.ValueType, encodedValue(k, v)), "status", "ok")
}
