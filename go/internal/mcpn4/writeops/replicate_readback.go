package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// replicaInverse removes each copy under its target and the name its reply assigned ("nn"),
// else the requested one.
func replicaInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(replicaData)
	out := []any{}
	for i := range replies {
		out = append(out, obj("nm", "v", "h", d.copies[i].parent, "n", replicaName(replies[i], d.copies[i].name)))
	}
	return out, nil
}

// replicaName is the "nn" of an add reply, else requested.
func replicaName(reply any, requested string) any {
	if first, ok := reply.([]any); ok && len(first) > 0 {
		if r, ok := first[0].(*pyjson.Object); ok {
			if v, ok := r.Get("nn"); ok {
				return v
			}
		}
	}
	return requested
}

// replicaReadback loads each target once and compares every copy's configured subtree with its
// planned body (copyDifferences, as n4_copy_component): "verified" only when every copy exists,
// has its source's type and holds exactly the planned configuration. observed lists, per copy, its
// target, name, type and the paths that differ; a copy noted as sharing a register whose address
// differs carries the cause.
func replicaReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(replicaData)
	loaded := map[string]*pyjson.Object{}
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i, r := range d.copies {
		nodes, ok := loaded[r.target]
		if !ok {
			var err error
			if _, nodes, err = plan.Handle(c, r.target, 1); err != nil {
				return nil, nil, nil, "", err
			}
			loaded[r.target] = nodes
		}
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		requested = append(requested, obj("target", r.target, "name", nn, "type", r.typ))
		var t any
		diffs := []any{"(copy not found)"}
		if n := box.Node(nodes, nn); n != nil {
			t = get(n, "t")
			var err error
			if diffs, err = copyDifferences(c, r.target, nn, r.body); err != nil {
				return nil, nil, nil, "", err
			}
		}
		entry := obj("target", r.target, "name", nn, "type", t, "differences", diffs)
		if text, ok := d.shared[i]; ok && touchesAddress(diffs) {
			entry.Set("cause", duplicateCause(text))
		}
		observed = append(observed, entry)
		if !box.PyEqual(t, r.typ) || len(diffs) > 0 {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}
