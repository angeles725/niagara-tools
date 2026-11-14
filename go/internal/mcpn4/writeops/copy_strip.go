package writeops

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// stripFlags are the slot flags a copy leaves out (baja Flags letters): r readonly, t transient,
// d default-on-clone (the platform's own clone skips such a frozen slot, ComplexSlotMap.newCopy)
// and R remove-on-clone (ComponentSlotMap.newCopy skips such a dynamic slot).
const stripFlags = "rtdR"

const notCopiedNote = "Not copied (flags r readonly, t transient, d default-on-clone, R remove-on-clone): "

// strippedNode is a node the copy leaves out: its path below the source and its effective flags;
// readonly marks a readonly-only slot holding a value off its declared default.
type strippedNode struct {
	path, flags string
	readonly    bool
}

// copyContracts holds the contracts of every type the copy evaluates (each node with children),
// read in one registry call; a client without the registry channel, or a type the station does
// not describe, is refused.
func copyContracts(c plan.Client, root *pyjson.Object) (*plan.Contracts, error) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, plan.ToolError("a copy needs the station's type definitions (registry channel) to " +
			"leave runtime slots out; this client cannot read them")
	}
	ct, specs := plan.NewContracts(tl), []string{}
	collectTypes(root, &specs)
	return ct, typeRefusal(ct.Load(specs))
}

func collectTypes(node *pyjson.Object, specs *[]string) {
	kids, _ := get(node, "s").([]any)
	if t, ok := get(node, "t").(string); ok && len(kids) > 0 {
		*specs = append(*specs, t)
	}
	for _, k := range kids {
		collectTypes(k.(*pyjson.Object), specs)
	}
}

func typeRefusal(err error) error {
	var u *plan.UnansweredType
	if errors.As(err, &u) {
		return plan.ToolError(u.Error() + "; a copy cannot tell its runtime slots from its configuration")
	}
	return err
}

// configured is the copy body of a loaded root: the nodes stripTree keeps, snapshot to CopyDepth
// (links recorded apart), without name-only children; with the nodes left out, in tree order.
func configured(ct *plan.Contracts, root *pyjson.Object) (*pyjson.Object, []strippedNode, *snapshot, error) {
	var out []strippedNode
	typ, _ := get(root, "t").(string)
	kept, err := stripTree(ct, root, typ, "", &out)
	if err != nil {
		return nil, nil, nil, typeRefusal(err)
	}
	s := &snapshot{handles: map[string]string{}}
	body, err := s.take(kept, "", CopyDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	return dropNameOnly(body), out, s, nil
}

// stripTree is node (of type typ) without each child whose effective flags hold one of
// stripFlags, with its subtree, at every level; children listed by name only are left out
// silently. A frozen child without "t" is evaluated as the type of its declared default.
func stripTree(ct *plan.Contracts, node *pyjson.Object, typ, path string, out *[]strippedNode) (*pyjson.Object, error) {
	res := pyjson.NewObject()
	for _, k := range node.Keys() {
		v, _ := node.Get(k)
		if k != "s" {
			res.Set(k, v)
			continue
		}
		kept := []any{}
		for _, c := range v.([]any) {
			kid := c.(*pyjson.Object)
			if nameOnly(kid) {
				continue
			}
			flags, slot, err := ct.Flags(typ, kid)
			if err != nil {
				return nil, err
			}
			p, def := joinPath(path, name(kid)), asObj(get(slot, "v"))
			if strings.ContainsAny(flags, stripFlags) {
				readonly := !strings.ContainsAny(flags, "tdR") && !sameValue(kid, def)
				*out = append(*out, strippedNode{p, flags, readonly})
				continue
			}
			kt, ok := get(kid, "t").(string)
			if !ok {
				kt, _ = get(def, "t").(string)
			}
			stripped, err := stripTree(ct, kid, kt, p, out)
			if err != nil {
				return nil, err
			}
			kept = append(kept, stripped)
		}
		if len(kept) > 0 {
			res.Set("s", kept)
		}
	}
	return res, nil
}

// sameValue is a node equal to the declared default: its value, and each child's by name.
func sameValue(node, def *pyjson.Object) bool {
	if def == nil || !box.PyEqual(get(node, "v"), get(def, "v")) {
		return false
	}
	names := map[string]bool{}
	for _, o := range []*pyjson.Object{node, def} {
		kids, _ := get(o, "s").([]any)
		for _, k := range kids {
			names[name(k.(*pyjson.Object))] = true
		}
	}
	for n := range names {
		a, b := kidNamed(node, n), kidNamed(def, n)
		if a == nil || b == nil || !sameValue(a, b) {
			return false
		}
	}
	return true
}

// strippedNotes is one note listing every stripped node, then one per readonly-only slot whose
// value is lost; none when nothing was stripped.
func strippedNotes(list []strippedNode) []any {
	if len(list) == 0 {
		return nil
	}
	items, notes := []string{}, []any{}
	for _, s := range list {
		items = append(items, s.path+" ("+s.flags+")")
	}
	notes = append(notes, notCopiedNote+strings.Join(items, ", "))
	for _, s := range list {
		if s.readonly {
			notes = append(notes, "readonly slot holds a non-default value and is not copied: "+s.path)
		}
	}
	return notes
}

// dropNameOnly is body without children left with no type, value or children, at every level:
// an add op child without a type is not a value, and the copy's type supplies those slots.
func dropNameOnly(body *pyjson.Object) *pyjson.Object {
	out := pyjson.NewObject()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		if k != "s" {
			out.Set(k, v)
			continue
		}
		kept := []any{}
		for _, c := range v.([]any) {
			if kid := dropNameOnly(c.(*pyjson.Object)); !nameOnly(kid) {
				kept = append(kept, kid)
			}
		}
		if len(kept) > 0 {
			out.Set("s", kept)
		}
	}
	return out
}

// nameOnly is a loaded child with no type, value or children: an action or a frozen slot at its
// default, listed by name only.
func nameOnly(kid *pyjson.Object) bool {
	return get(kid, "t") == nil && get(kid, "v") == nil && get(kid, "s") == nil
}

func joinPath(path, n string) string {
	if path == "" {
		return n
	}
	return path + "/" + n
}

func asObj(v any) *pyjson.Object { o, _ := v.(*pyjson.Object); return o }

// kidNamed is node's child named n; nil when absent.
func kidNamed(node *pyjson.Object, n string) *pyjson.Object {
	kids, _ := get(node, "s").([]any)
	for _, k := range kids {
		if o, ok := k.(*pyjson.Object); ok && name(o) == n {
			return o
		}
	}
	return nil
}
