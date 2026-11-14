package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A copy never loses a source child in silence: every child the station loaded is in the add op
// with the flags the source has, or is an entry of the plan's "Not copied" note (it or an
// ancestor), or bare (name only: no type, value, children or flags, so the copy's type supplies it). The lost-flags defect found on a live 4.15 station was a
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

// notCopied is the set of paths the plan's "Not copied (...): path (flags), ..." note lists.
func notCopied(notes []any) map[string]bool {
	listed := map[string]bool{}
	for _, n := range notes {
		text, _ := n.(string)
		_, items, found := strings.Cut(text, "): ")
		if !strings.HasPrefix(text, "Not copied (") || !found {
			continue
		}
		for _, item := range strings.Split(items, ", ") {
			if at := strings.LastIndex(item, " ("); at > 0 {
				listed[item[:at]] = true
			}
		}
	}
	return listed
}

// unaccounted lists the source children that are neither copied with their flags, listed nor bare.
func unaccounted(source, body *pyjson.Object, path string, listed map[string]bool, out *[]string) {
	for _, kid := range kidsOfNode(source) {
		p := nodeName(kid)
		if path != "" {
			p = path + "/" + p
		}
		if listed[p] || bare(kid) {
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
		if want, got := flagsOf(kid), flagsOf(copied); want != got {
			*out = append(*out, p+" (flags "+got+", source "+want+")")
		}
		unaccounted(kid, copied, p, listed, out)
	}
}

func flagsOf(node *pyjson.Object) string { f, _ := node.Get("f"); s, _ := f.(string); return s }

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
		p, _, _ := copyPlanOf(t, tree)
		if len(p.Ops) == 0 {
			t.Errorf("%s: the plan has no op", label)
			continue
		}
		op, _ := p.Ops[0].(*pyjson.Object)
		var body *pyjson.Object
		if op != nil {
			b, _ := op.Get("b")
			body, _ = b.(*pyjson.Object)
		}
		if body == nil {
			t.Errorf("%s: the add op has no body", label)
			continue
		}
		var lost []string
		unaccounted(tree, body, "", notCopied(p.Notes), &lost)
		if len(lost) > 0 {
			t.Errorf("%s: neither copied, listed nor bare: %v", label, lost)
		}
	}
}

// The check itself is pinned: a body that drops a child, or keeps it without the source's flags,
// is reported; an entry of the Not copied note and a bare child are not.
func TestTheAccountingCheckReportsALostChildAndALostFlag(t *testing.T) {
	source := obj("s", []any{
		obj("n", "kept", "t", "baja:String", "v", "x", "f", "h"),
		obj("n", "gone", "t", "baja:String", "v", "y"),
		obj("n", "listed", "t", "baja:String", "v", "z", "f", "rt"),
		obj("n", "plain"),
	})
	body := obj("s", []any{obj("n", "kept", "t", "baja:String", "v", "x")})
	var lost []string
	unaccounted(source, body, "", notCopied([]any{"Not copied (flags r readonly): listed (rt)"}), &lost)
	if got := strings.Join(lost, "; "); got != "kept (flags , source h); gone" {
		t.Errorf("reported %q, want the lost flag of kept and the lost child gone", got)
	}
}
