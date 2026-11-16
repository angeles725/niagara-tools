package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// addAlarmInverse removes each extension whose add was sent, under the name the station
// assigned: the same rule as n4_add_history's inverse, which it reuses.
func addAlarmInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(alarmData)
	return addHistoryInverse(&plan.Planned{Ops: p.Ops, Data: historyData{parents: d.parents, adds: d.adds}}, replies)
}

// alarmReadDepth reaches extension/offnormalAlgorithm/limitEnable/highLimitEnable under a point.
const alarmReadDepth = 4

// addAlarmReadback reads every slot the batch set on each extension and the type of both
// algorithms. A slot the station stored differently lands in coerced ({slot: {sent, read}});
// the verdict is mismatch when an extension is missing, its type or an algorithm's type is not
// the requested one, or any slot was coerced.
func addAlarmReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(alarmData)
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, pt := range d.points {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		want := obj("ord", pt.ord, "name", nn, "type", alarmExtType, "offnormalAlgorithm", pt.kind.offnormal,
			"faultAlgorithm", orNil(pt.kind.fault), "slots", alarmSlotMap(pt.writes, func(w alarmWrite) any { return w.want }))
		requested = append(requested, want)
		got, ok, err := observeAlarm(c, pt, nn, want)
		if err != nil {
			return nil, nil, nil, "", err
		}
		observed = append(observed, got)
		if !ok {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

func alarmSlotMap(ws []alarmWrite, value func(alarmWrite) any) *pyjson.Object {
	out := pyjson.NewObject()
	for _, w := range ws {
		out.Set(w.path, value(w))
	}
	return out
}

// observeAlarm reads one extension back: ok is false when it is missing, of another type, an
// algorithm is of another type or a slot reads other than sent.
func observeAlarm(c plan.Client, pt alarmPoint, nn string, want *pyjson.Object) (any, bool, error) {
	nodes, err := c.LoadTree(pt.ord, alarmReadDepth, box.LoadOptions{})
	if err != nil {
		return nil, false, err
	}
	n := box.Node(nodes, nn)
	if n == nil {
		return nil, false, nil
	}
	out := obj("ord", pt.ord, "name", nn, "type", get(n, "t"),
		"offnormalAlgorithm", get(box.Node(nodes, nn+"/offnormalAlgorithm"), "t"))
	out.Set("faultAlgorithm", nil)
	if pt.kind.fault != "" {
		out.Set("faultAlgorithm", get(box.Node(nodes, nn+"/faultAlgorithm"), "t"))
	}
	ok := true
	for _, k := range []string{"type", "offnormalAlgorithm", "faultAlgorithm"} {
		w, _ := want.Get(k)
		g, _ := out.Get(k)
		ok = ok && box.PyEqual(g, w)
	}
	slots, coerced := pyjson.NewObject(), pyjson.NewObject()
	for _, w := range pt.writes {
		v, err := observeAlarmSlot(c, &nodes, nn+"/"+w.path, w.vt)
		if err != nil {
			return nil, false, err
		}
		slots.Set(w.path, v)
		if !box.PyEqual(v, w.want) {
			coerced.Set(w.path, obj("sent", w.sent, "read", v))
		}
	}
	out.Set("slots", slots)
	if len(coerced.Keys()) > 0 {
		out.Set("coerced", coerced)
		ok = false
	}
	return out, ok, nil
}

// observeAlarmSlot is the value of path in plan.Observe's form; a slot at its type default is
// filled from the type definition (plan.Resolve), and an omitted baja:StatusBoolean (the station
// leaves a default value out) reads {value: false, status: "0"}.
func observeAlarmSlot(c plan.Client, nodes **pyjson.Object, path, vt string) (any, error) {
	var err error
	if *nodes, err = plan.Resolve(c, *nodes, path); err != nil {
		return nil, err
	}
	switch vt {
	case "baja:StatusBoolean":
		if box.Node(*nodes, path) == nil {
			return obj("value", false, "status", "0"), nil
		}
	case "baja:Double", "baja:Boolean", "baja:Format", "baja:RelTime", "baja:String":
	default:
		return plan.Observe(*nodes, path, "baja:String")
	}
	return plan.Observe(*nodes, path, vt)
}
