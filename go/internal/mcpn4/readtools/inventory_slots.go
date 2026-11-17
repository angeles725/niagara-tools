package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The slots an inventory snapshot records for one component. Configured properties are the simple
// values and Status values (a Status as "value {status}") directly on it, leaving out runtime state
// so two snapshots of an unchanged station diff to empty. The type CONTRACT decides, not the slot
// type: a Status slot can be configuration (BNumericWritable.fallback, flags 0). Left out:
//   - a slot flagged t transient on the instance or in its contract (BNumericPoint.out 267 = rtso,
//     BNumericWritable inputs other than in1 and in8 transient, BHistoryExt.status and lastRecord 3 = rt);
//   - a Status out, unless its contract declares it without r and t (a constant block's configured
//     out): writeops' keepsOutput rule, so an out stays out when the contracts are not read;
//   - an input a link drives (a link child's targetSlotName): its source sets it, so it changes
//     with the source (in1 and in8 are readonly, not transient, so only this rule catches them).
// Readonly alone is configuration (in1/in8 set by the emergency and manual actions persist).

// runtimeFlag is the slot flag that marks state the station never persists; computedOutFlags
// are the declared flags that make an out computed (writeops computedFlags).
const (
	runtimeFlag      = "t"
	computedOutFlags = "rt"
)

// slots are the recorded slots of the component at path, in load order.
func (r *invReader) slots(path string) *pyjson.Object {
	out := pyjson.NewObject()
	node, kids := box.Node(r.nodes, path), box.Children(r.nodes, path)
	if r.chosen != nil {
		for _, name := range r.chosen {
			out.Set(name, r.slotValue(path, name, box.Node(kids, name)))
		}
		return out
	}
	driven := drivenSlots(kids)
	for _, name := range kids.Keys() {
		kid := box.Node(kids, name)
		if driven[name] || r.runtime(node, kid) || name == "out" && r.computedOut(node, kid) {
			continue
		}
		if v := r.slotValue(path, name, kid); v != nil {
			out.Set(name, v)
		}
	}
	return out
}

// slotValue is the value text of a slot, a Status value as "value {status}"; nil for a component,
// a link, a struct of another kind, or a slot the load does not hold (at its default, or none).
func (r *invReader) slotValue(path, name string, kid *pyjson.Object) any {
	if kid == nil {
		return nil
	}
	if v, ok := kid.Get("v"); ok {
		return valueText(v)
	}
	typ := valueText(lookup(kid, "t"))
	if !strings.HasPrefix(typ, "baja:Status") {
		return nil
	}
	full := childPath(path, name)
	value, status, err := box.StatusValue(r.nodes, full)
	if err != nil { // StatusEnum, StatusString: the children as loaded
		kids := box.Children(r.nodes, full)
		value, status = box.ChildValue(kids, "value", nil), box.ChildValue(kids, "status", "0")
	}
	return valueText(value) + " {" + valueText(status) + "}"
}

// drivenSlots are the slots of a component its links drive (their targetSlotName).
func drivenSlots(kids *pyjson.Object) map[string]bool {
	out := map[string]bool{}
	for _, name := range kids.Keys() {
		if !isLinkType(lookup(box.Node(kids, name), "t")) {
			continue
		}
		for _, k := range kidsOf(box.Node(kids, name)) {
			if lookup(k, "n") == "targetSlotName" {
				out[valueText(lookup(k, "v"))] = true
			}
		}
	}
	return out
}

// kidsOf is a loaded node's "s" children (LoadTree checked them: a list of named objects).
func kidsOf(node *pyjson.Object) []*pyjson.Object {
	raw, _ := lookup(node, "s").([]any)
	out := make([]*pyjson.Object, 0, len(raw))
	for _, k := range raw {
		if o, ok := k.(*pyjson.Object); ok {
			out = append(out, o)
		}
	}
	return out
}

// runtime says whether kid, a slot of node, is flagged transient on the instance or in its contract.
func (r *invReader) runtime(node, kid *pyjson.Object) bool {
	if f, _ := lookup(kid, "f").(string); strings.Contains(f, runtimeFlag) {
		return true
	}
	f, ok := r.declaredFlags(node, kid)
	return ok && strings.Contains(f, runtimeFlag)
}

// computedOut says whether kid, an out slot of node, is a computed Status output: a Status out
// whose contract does not declare it without r and t.
func (r *invReader) computedOut(node, kid *pyjson.Object) bool {
	if !strings.HasPrefix(valueText(lookup(kid, "t")), "baja:Status") {
		return false
	}
	f, ok := r.declaredFlags(node, kid)
	return !ok || strings.ContainsAny(f, computedOutFlags)
}

// declaredFlags are the contract flags of kid on node's type; ok is false when the contracts were
// not read, or they do not declare kid (a dynamic slot, or a type they do not describe).
func (r *invReader) declaredFlags(node, kid *pyjson.Object) (string, bool) {
	if r.ct == nil || node == nil {
		return "", false
	}
	typ, _ := lookup(node, "t").(string)
	slot, err := r.ct.Declared(typ, kid)
	if err != nil || slot == nil {
		return "", false
	}
	f, _ := lookup(slot, "f").(string)
	return f, true
}
