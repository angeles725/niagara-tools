package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
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

// copyReadback loads every copy under its assigned name, CopyDepth deep, and compares its
// configured subtree (the same strip and snapshot as the plan) with the planned body, ignoring
// child order and handles: "verified" only when each copy exists, has the source type and holds
// exactly the planned configuration. observed lists, per copy, the paths that differ or are missing.
func copyReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(copyData)
	parentOrd := str(args, "parent_ord")
	_, nodes, err := plan.Handle(c, parentOrd, 1)
	if err != nil {
		return nil, nil, nil, "", err
	}
	requested, observed, verdict := []any{}, []any{}, "verified"
	for i := range d.names {
		nn, _ := get(inverse[i].(*pyjson.Object), "n").(string)
		requested = append(requested, obj("name", nn, "type", d.sourceType))
		var t any
		diffs := []any{"(copy not found)"}
		if n := box.Node(nodes, nn); n != nil {
			t = get(n, "t")
			if diffs, err = copyDifferences(c, parentOrd, nn, d.body); err != nil {
				return nil, nil, nil, "", err
			}
		}
		observed = append(observed, obj("name", nn, "type", t, "differences", diffs))
		if !box.PyEqual(t, d.sourceType) || len(diffs) > 0 {
			verdict = "mismatch"
		}
	}
	return requested, replies, observed, verdict, nil
}

// copyDifferences loads copy nn and lists the paths where its configured subtree and want differ.
func copyDifferences(c plan.Client, parentOrd, nn string, want *pyjson.Object) ([]any, error) {
	copyOrd, err := ord.ChildOrd(parentOrd, nn)
	if err != nil {
		return nil, err
	}
	nodes, err := c.LoadTree(copyOrd, CopyDepth, box.LoadOptions{})
	if err != nil {
		return nil, err
	}
	root := box.Node(nodes, "")
	ct, err := copyContracts(c, root)
	if err != nil {
		return nil, err
	}
	got, _, _, err := configured(ct, root)
	if err != nil {
		return nil, err
	}
	out := []any{}
	diffTree(want, got, "", &out)
	return out, nil
}

// diffTree appends the path of each node whose type or value differs between want and got, of
// each child of want that got lacks and of each child got holds beyond want, matched by name.
func diffTree(want, got *pyjson.Object, path string, out *[]any) {
	if path != "" && (!box.PyEqual(get(want, "t"), get(got, "t")) || !box.PyEqual(get(want, "v"), get(got, "v"))) {
		*out = append(*out, path)
		return
	}
	wantKids, _ := get(want, "s").([]any)
	for _, k := range wantKids {
		w := k.(*pyjson.Object)
		if g := kidNamed(got, name(w)); g != nil {
			diffTree(w, g, joinPath(path, name(w)), out)
		} else {
			*out = append(*out, joinPath(path, name(w)))
		}
	}
	gotKids, _ := get(got, "s").([]any)
	for _, k := range gotKids {
		if g := k.(*pyjson.Object); kidNamed(want, name(g)) == nil {
			*out = append(*out, joinPath(path, name(g)))
		}
	}
}
