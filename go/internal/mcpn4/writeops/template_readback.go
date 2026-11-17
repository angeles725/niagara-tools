package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/ord"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// deployInverse removes the deployed component under the name its reply assigned ("nn"), else
// the requested one.
func deployInverse(p *plan.Planned, replies []any) ([]any, error) {
	d := p.Data.(deployData)
	nn := any(d.name)
	if first, ok := replies[0].([]any); ok && len(first) > 0 {
		if reply, ok := first[0].(*pyjson.Object); ok {
			if v, ok := reply.Get("nn"); ok {
				nn = v
			}
		}
	}
	return []any{obj("nm", "v", "h", d.parent, "n", nn)}, nil
}

// deployUnapplied is true when the parent has no child it did not have when planned.
func deployUnapplied(c plan.Client, args *pyjson.Object, p *plan.Planned) (bool, error) {
	_, nodes, err := plan.Handle(c, str(args, "parent_ord"), 1)
	if err != nil {
		return false, err
	}
	for n := range childNames(nodes) {
		if !p.Data.(deployData).children[n] {
			return false, nil
		}
	}
	return true, nil
}

// deployReadback loads the deployed component as deep as the body and checks every planned node:
// "verified" only when each exists with its planned type, value and flags. A planned slot the
// station does not list is accepted when it equals the declared default (a station lists a
// frozen slot only off its default). Slots the station adds (status, out...) are not compared.
func deployReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, inverse []any) (any, any, any, string, error) {
	d := p.Data.(deployData)
	inv, _ := inverse[0].(*pyjson.Object)
	nn, _ := get(inv, "n").(string)
	requested := obj("name", nn, "type", get(d.body, "t"))
	childOrd, err := ord.ChildOrd(str(args, "parent_ord"), nn)
	if err != nil {
		return nil, nil, nil, "", err
	}
	nodes, err := c.LoadTree(childOrd, tplDepth(d.body), box.LoadOptions{})
	var root *pyjson.Object
	if err == nil {
		root = box.Node(nodes, "")
	}
	if root == nil {
		return requested, replies, obj("name", nn, "type", nil, "differences", []any{"(component not found)"}), "mismatch", nil
	}
	tl, _ := c.(plan.TypeLoader)
	diffs := []any{}
	if !box.PyEqual(get(root, "t"), get(d.body, "t")) {
		diffs = append(diffs, "(root type)")
	}
	tplDiff(tl, d.body, root, "", &diffs)
	verdict := "verified"
	if len(diffs) > 0 {
		verdict = "mismatch"
	}
	return requested, replies, obj("name", nn, "type", get(root, "t"), "differences", diffs), verdict, nil
}

// tplDiff appends the path of each planned child of want that got lacks (and is not at its
// declared default) or holds with another type, value or flags.
func tplDiff(tl plan.TypeLoader, want, got *pyjson.Object, path string, out *[]any) {
	parentType, _ := get(want, "t").(string)
	for _, w := range objects(get(want, "s")) {
		p := joinPath(path, name(w))
		g := kidNamed(got, name(w))
		if g == nil {
			if !tplAtDefault(tl, parentType, w) {
				*out = append(*out, p)
			}
			continue
		}
		if !tplSame(w, g) {
			*out = append(*out, p)
			continue
		}
		tplDiff(tl, w, g, p, out)
	}
}

// tplSame compares the type, value and flags a planned node states; a status value is compared
// up to its facets (";activeLevel=..."), which the station recomputes.
func tplSame(w, g *pyjson.Object) bool {
	for _, k := range []string{"t", "f"} {
		if v := get(w, k); v != nil && !box.PyEqual(v, get(g, k)) {
			return false
		}
	}
	wv, has := w.Get("v")
	if !has || box.PyEqual(wv, get(g, "v")) {
		return true
	}
	ws, ok1 := wv.(string)
	gs, ok2 := get(g, "v").(string)
	status := get(w, "t") == "baja:Status" || name(w) == "status"
	return status && ok1 && ok2 && strings.SplitN(ws, ";", 2)[0] == strings.SplitN(gs, ";", 2)[0]
}

// tplAtDefault is a planned frozen slot equal to what parentType declares for it.
func tplAtDefault(tl plan.TypeLoader, parentType string, w *pyjson.Object) bool {
	if tl == nil || parentType == "" {
		return false
	}
	def, err := plan.DeclaredDefault(tl, parentType, name(w))
	if err != nil || def == nil {
		return false
	}
	if !tplSame(w, def) {
		return false
	}
	typed := pyjson.NewObject() // w under the declared type, for its children's defaults
	for _, k := range w.Keys() {
		v, _ := w.Get(k)
		typed.Set(k, v)
	}
	if get(typed, "t") == nil {
		typed.Set("t", get(def, "t"))
	}
	missing := []any{}
	tplDiff(tl, typed, def, "", &missing)
	return len(missing) == 0
}

// tplDepth is how many levels of children a body holds.
func tplDepth(body *pyjson.Object) int {
	deepest := 0
	for _, k := range objects(get(body, "s")) {
		deepest = max(deepest, 1+tplDepth(k))
	}
	return deepest
}
