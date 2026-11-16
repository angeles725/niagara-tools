package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// addHistoryInverse removes each extension whose add was sent, under the name its reply
// assigned ("nn"), else the requested one; a capacity set op needs no undo of its own.
func addHistoryInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(historyData)
	out := []any{}
	for i, at := range d.adds {
		if at >= len(replies) {
			break
		}
		nn := get(p.Ops[at].(*pyjson.Object), "n")
		if first, ok := replies[at].([]any); ok && len(first) > 0 {
			if reply, ok := first[0].(*pyjson.Object); ok {
				if v, ok := reply.Get("nn"); ok {
					nn = v
				}
			}
		}
		out = append(out, obj("nm", "v", "h", d.parents[i], "n", nn))
	}
	return out, nil
}

// addHistoryReadback compares each extension with what was asked: its type, configured slots
// (no interval for a change-of-value one), the capacity, full policy and change tolerance when
// asked, the history id the
// dry run predicted and no fault, so an extension in fault "Duplicate history id" reads
// mismatch. A nil requested field (an id the pattern could not predict) is not checked.
func addHistoryReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(historyData)
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, s := range d.requested {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		want := obj("ord", s.ord, "name", nn, "type", s.typ, "historyName", s.pattern)
		if s.mode == "Interval" {
			want.Set("interval", s.interval)
		}
		want.Set("enabled", s.enabled)
		if s.capacity != "" {
			want.Set("capacity", s.capacity)
		}
		if s.fullPolicy != "" {
			want.Set("fullPolicy", s.fullPolicy)
		}
		if s.tolerance != nil {
			want.Set("changeTolerance", s.tolerance)
		}
		want.Set("historyId", orNil(s.id))
		want.Set("faultCause", "")
		requested = append(requested, want)
		got, err := observeHistory(c, s.ord, nn, want)
		if err != nil {
			return nil, nil, nil, "", err
		}
		observed = append(observed, got)
		if !historyMatches(want, got) {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

func orNil(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// historyMatches is whether got holds every requested field that is not nil (unchecked).
func historyMatches(want *pyjson.Object, got any) bool {
	g, ok := got.(*pyjson.Object)
	if !ok {
		return false
	}
	for _, k := range want.Keys() {
		w, _ := want.Get(k)
		if v, _ := g.Get(k); w != nil && !box.PyEqual(v, w) {
			return false
		}
	}
	return true
}

// historyFields are the read-back fields: the name in the result, the slot path under the
// extension and its value type.
var historyFields = []struct{ key, slot, vt string }{
	{"historyName", "historyName", "baja:Format"}, {"interval", "interval", "baja:RelTime"},
	{"enabled", "enabled", "baja:Boolean"}, {"capacity", "historyConfig/capacity", "history:Capacity"},
	{"historyId", "historyConfig/id", "history:HistoryId"}, {"faultCause", "faultCause", "baja:String"},
	{"fullPolicy", "historyConfig/fullPolicy", "history:FullPolicy"},
	{"changeTolerance", "changeTolerance", "baja:Double"},
}

// observeHistory reads back the fields of want of one extension; a slot at its type default is
// filled from the type definition (plan.Resolve), so a requested default value still compares
// equal.
func observeHistory(c plan.Client, ord, nn string, want *pyjson.Object) (any, error) {
	nodes, err := c.LoadTree(ord, 3, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	n := box.Node(nodes, nn)
	if n == nil {
		return nil, nil
	}
	out := obj("ord", ord, "name", nn, "type", get(n, "t"))
	for _, f := range historyFields {
		if _, asked := want.Get(f.key); !asked {
			continue
		}
		path := nn + "/" + f.slot
		if nodes, err = plan.Resolve(c, nodes, path); err != nil {
			return nil, err
		}
		v, err := plan.Observe(nodes, path, f.vt)
		if err != nil {
			return nil, err
		}
		out.Set(f.key, v)
	}
	return out, nil
}
