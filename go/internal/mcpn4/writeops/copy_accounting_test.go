package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A copy never loses a source child in silence: every child the station loaded is in the add op,
// or named in the plan notes (it or an ancestor), or bare (name only: no type, value, children or
// flags, so the copy's type supplies it). The lost-flags defect found on a live 4.15 station was a
// child in none of the three.

func kidsOfNode(node *pyjson.Object) []*pyjson.Object {
	raw, _ := node.Get("s")
	list, _ := raw.([]any)
	out := []*pyjson.Object{}
	for _, k := range list {
		if o, ok := k.(*pyjson.Object); ok {
			out = append(out, o)
		}
	}
	return out
}

func nodeName(node *pyjson.Object) string { n, _ := node.Get("n"); s, _ := n.(string); return s }

func bare(node *pyjson.Object) bool {
	for _, k := range []string{"t", "v", "s", "f"} {
		if _, has := node.Get(k); has {
			return false
		}
	}
	return true
}

func namedKid(node *pyjson.Object, n string) *pyjson.Object {
	for _, k := range kidsOfNode(node) {
		if nodeName(k) == n {
			return k
		}
	}
	return nil
}

// unaccounted lists the paths of source children that are neither copied, listed nor bare.
func unaccounted(source, body *pyjson.Object, path, notes string, out *[]string) {
	for _, kid := range kidsOfNode(source) {
		p := nodeName(kid)
		if path != "" {
			p = path + "/" + p
		}
		if strings.Contains(notes, p+" (") || bare(kid) {
			continue
		}
		var copied *pyjson.Object
		if body != nil {
			copied = namedKid(body, nodeName(kid))
		}
		if copied == nil {
			*out = append(*out, p)
			continue
		}
		unaccounted(kid, copied, p, notes, out)
	}
}

func TestACopyAccountsForEverySourceChild(t *testing.T) {
	shapes := map[string]func(tree *pyjson.Object){
		"the device as loaded": func(*pyjson.Object) {},
		"a frozen slot at its default value with a flag": func(tree *pyjson.Object) {
			addKids(tree, obj("nm", "p", "n", "enabled", "f", "h"))
		},
		"a frozen slot with a value and a flag": func(tree *pyjson.Object) {
			addKids(tree, obj("nm", "p", "n", "enabled", "t", "baja:Boolean", "v", "false", "f", "o"))
		},
		"a flagged dynamic folder with a flagged child": func(tree *pyjson.Object) {
			addKids(tree, obj("nm", "p", "n", "Cfg", "t", "baja:Folder", "f", "h", "s", []any{
				obj("nm", "p", "n", "limit", "t", "baja:Double", "v", "5.0", "f", "rh")}))
		},
		"a dynamic slot removed on clone": func(tree *pyjson.Object) {
			addKids(tree, obj("nm", "p", "n", "Tag", "t", "baja:String", "v", "x", "f", "R"))
		},
		"a flag deep in the tree": func(tree *pyjson.Object) {
			addKids(under(tree, "points"), obj("nm", "p", "n", "Note", "t", "baja:String", "v", "n", "f", "s"))
		},
	}
	for label, shape := range shapes {
		tree := gwTree()
		shape(tree)
		p, _, notes := copyPlanOf(t, tree)
		op, _ := p.Ops[0].(*pyjson.Object)
		b, _ := op.Get("b")
		body, _ := b.(*pyjson.Object)
		var lost []string
		unaccounted(tree, body, "", notes, &lost)
		if len(lost) > 0 {
			t.Errorf("%s: neither copied, listed nor bare: %v", label, lost)
		}
	}
}
