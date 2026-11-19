package writeops

import (
	"errors"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// What a copy leaves out, by slot flags (baja Flags letters). A flag an integrator sets is
// configuration and is copied with the slot; only runtime state is left out:
//   - a frozen slot by its DECLARED flags: r readonly, t transient, d default-on-clone (the
//     platform's own clone skips a frozen d slot, ComplexSlotMap.newCopy); also when its instance
//     flags hold t, since a transient slot is never persisted;
//   - a dynamic slot only by its instance flags t or R remove-on-clone (ComponentSlotMap.newCopy
//     skips a dynamic R slot).
const (
	frozenStrip  = "rtd"
	dynamicStrip = "tR"
)

// copyContracts holds the contracts of every type the copy evaluates (each node with children),
// read in one registry call; a client without the registry channel, a type the station does not
// describe or a malformed child is refused.
func copyContracts(c plan.Client, root *pyjson.Object) (*plan.Contracts, error) {
	tl, ok := c.(plan.TypeLoader)
	if !ok {
		return nil, plan.ToolError("a copy needs the station's type definitions (registry channel) to " +
			"leave runtime slots out; this client cannot read them")
	}
	ct, err := treeContracts(tl, root)
	return ct, typeRefusal(err)
}

// treeContracts reads the contracts of every type in root that has children in one registry
// call; the error is Load's own (*plan.UnansweredType for a type the station does not describe).
func treeContracts(tl plan.TypeLoader, root *pyjson.Object) (*plan.Contracts, error) {
	ct, specs := plan.NewContracts(tl), []string{}
	if err := collectTypes(root, "", &specs); err != nil {
		return nil, err
	}
	return ct, ct.Load(specs)
}

func collectTypes(node *pyjson.Object, path string, specs *[]string) error {
	kids, err := childObjects(node, path)
	if err != nil {
		return err
	}
	if t, ok := get(node, "t").(string); ok && len(kids) > 0 {
		*specs = append(*specs, t)
	}
	for _, k := range kids {
		if err := collectTypes(k, joinPath(path, name(k)), specs); err != nil {
			return err
		}
	}
	return nil
}

func typeRefusal(err error) error {
	var u *plan.UnansweredType
	if errors.As(err, &u) {
		return plan.ToolError(u.Error() + "; a copy cannot tell its runtime slots from its configuration")
	}
	return err
}

// configured is the copy body of a loaded root: the nodes stripTree keeps, snapshot to CopyDepth
// (links recorded apart), without name-only children, each kept node carrying its own flags;
// with the nodes left out, in tree order.
func configured(ct *plan.Contracts, root *pyjson.Object) (*pyjson.Object, []strippedNode, *snapshot, error) {
	var out []strippedNode
	typ, _ := get(root, "t").(string)
	kept, err := stripTree(ct, root, typ, "", &out)
	if err != nil {
		return nil, nil, nil, typeRefusal(err)
	}
	s := &snapshot{handles: map[string]string{}, ct: ct}
	body, err := s.take(kept, "", CopyDepth)
	if err != nil {
		return nil, nil, nil, err
	}
	carryFlags(kept, body) // before dropNameOnly: a slot kept for its flags alone is not name-only
	return dropNameOnly(body), out, s, nil
}

// stripTree is node (of type typ) without each child stripReason leaves out, with its subtree,
// at every level; children listed by name only are left out silently. A frozen child without "t"
// is evaluated as the type of its declared default. The status of a Status value off its default
// is kept whatever its flags (carriedStatus).
func stripTree(ct *plan.Contracts, node *pyjson.Object, typ, path string, out *[]strippedNode) (*pyjson.Object, error) {
	kids, err := childObjects(node, path)
	if err != nil {
		return nil, err
	}
	res := pyjson.NewObject()
	for _, k := range node.Keys() {
		if v, _ := node.Get(k); k != "s" {
			res.Set(k, v)
		}
	}
	kept := []any{}
	for _, kid := range kids {
		if nameOnly(kid) {
			continue
		}
		slot, err := ct.Declared(typ, kid)
		if err != nil {
			return nil, err
		}
		p, def := joinPath(path, name(kid)), asObj(get(slot, "v"))
		carried := carriedStatus(typ, kid, def)
		if flags, strip, readonly := stripReason(kid, slot); strip && !carried {
			*out = append(*out, strippedNode{p, flags, readonly && !sameValue(kid, def)})
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
		if carried {
			stripped = asStatusChild(stripped)
		}
		kept = append(kept, stripped)
	}
	if len(kept) > 0 {
		res.Set("s", kept)
	}
	return res, nil
}

// stripReason says whether kid (declared by slot, nil for a dynamic slot) is left out, the flags
// that decided it, and whether it is a frozen slot declared readonly-only (its value is lost).
func stripReason(kid, slot *pyjson.Object) (flags string, strip, readonly bool) {
	instance, _ := get(kid, "f").(string) // an absent "f" is ""
	if slot == nil {
		return instance, strings.ContainsAny(instance, dynamicStrip), false
	}
	declared, _ := get(slot, "f").(string)
	if strings.ContainsAny(declared, frozenStrip) {
		readonly = !strings.ContainsAny(declared, "td") && !strings.Contains(instance, "t")
		return declared, true, readonly
	}
	if strings.Contains(instance, "t") {
		return instance, true, false
	}
	return "", false, false
}
