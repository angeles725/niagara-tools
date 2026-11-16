package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// addHistoryInverse removes each extension under the name its reply assigned ("nn"), else the
// requested one.
func addHistoryInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(historyData)
	out := []any{}
	for i := range replies {
		nn := get(p.Ops[i].(*pyjson.Object), "n")
		if first, ok := replies[i].([]any); ok && len(first) > 0 {
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

func addHistoryReadback(c plan.Client, _ *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(historyData)
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, s := range d.requested {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		want := obj("ord", s.ord, "name", nn, "type", s.typ, "historyName", s.pattern,
			"interval", s.interval, "enabled", s.enabled)
		requested = append(requested, want)
		got, err := observeHistory(c, s.ord, nn)
		if err != nil {
			return nil, nil, nil, "", err
		}
		observed = append(observed, got)
		if !box.PyEqual(got, want) {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

// observeHistory reads one extension back; a slot at its type default is filled from the type
// definition (plan.Resolve), so a requested default value still compares equal.
func observeHistory(c plan.Client, ord, nn string) (any, error) {
	nodes, err := c.LoadTree(ord, 3, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	n := box.Node(nodes, nn)
	if n == nil {
		return nil, nil
	}
	out := obj("ord", ord, "name", nn, "type", get(n, "t"))
	for _, f := range []struct{ slot, vt string }{{"historyName", "baja:Format"},
		{"interval", "baja:RelTime"}, {"enabled", "baja:Boolean"}} {
		path := nn + "/" + f.slot
		if nodes, err = plan.Resolve(c, nodes, path); err != nil {
			return nil, err
		}
		v, err := plan.Observe(nodes, path, f.vt)
		if err != nil {
			return nil, err
		}
		out.Set(f.slot, v)
	}
	return out, nil
}
