package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// copyInverse removes each copy under the name its reply assigned ("nn"), else the requested one.
func copyInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(copyData)
	out := []any{}
	for i := range replies {
		nn := any(d.names[i])
		if first, ok := replies[i].([]any); ok && len(first) > 0 {
			if reply, ok := first[0].(*pyjson.Object); ok {
				if v, ok := reply.Get("nn"); ok {
					nn = v
				}
			}
		}
		out = append(out, obj("nm", "v", "h", d.parent, "n", nn))
	}
	return out, nil
}

// copyReadback finds every copy under its assigned name with the source's type.
func copyReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(copyData)
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i := range d.names {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		requested = append(requested, obj("name", nn, "type", d.sourceType))
		var t any
		if n := box.Node(nodes, nn); n != nil {
			t = get(n, "t")
		}
		observed = append(observed, obj("name", nn, "type", t))
		if !box.PyEqual(t, d.sourceType) {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}
