package readtools

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The station read of n4_inventory_snapshot: one bounded BOX load of the subtree, then every
// component down to depth levels with the slots it records.
//
// Configured properties are the simple values ("v") directly on a component, leaving out runtime
// state so two snapshots of an unchanged station diff to empty: every Status value (an out, an
// input, a status) and every slot flagged t transient, on the instance or in the type's contract
// (writeops' runtime-slot rule). Readonly alone is configuration, as there.

// runtimeFlag is the slot flag that marks state the station never persists.
const runtimeFlag = "t"

// invReader is one snapshot read: the load, the contracts (nil: instance flags only) and the
// slots argument (nil: configured properties).
type invReader struct {
	nodes  *pyjson.Object
	ct     *plan.Contracts
	chosen []string
}

// readInventory loads base and returns its components down to depth, at most maxComponents.
func (t *Tools) readInventory(base string, depth, maxComponents int, chosen []string) (*invSnapshot, []string, error) {
	extra := 1 // the properties of the deepest components
	if chosen != nil {
		extra = 2 // and the value and status of a chosen Status slot
	}
	nodes, _, err := t.timedLoad(base, depth+extra)
	if err != nil {
		return nil, nil, err
	}
	r := &invReader{nodes: nodes, chosen: chosen}
	notes := []string{}
	if chosen == nil {
		if note := r.loadContracts(t, depth); note != "" {
			notes = append(notes, t.Ctx.Scrub(note))
		}
	}
	s := &invSnapshot{base: base, depth: depth, contracts: r.ct != nil, chosen: chosen}
	kept := map[string]bool{}
	for _, path := range nodes.Keys() {
		node := box.Node(nodes, path)
		if path != "" && (!kept[parentPath(path)] || strings.Count(path, "/") >= depth) {
			continue
		}
		if yes, err := isComponent(node); path != "" && (err != nil || !yes) {
			continue
		}
		if len(s.components) == maxComponents {
			s.truncated = true
			break
		}
		kept[path] = true
		s.components = append(s.components, invComponent{path: invSlotPath(base, path),
			typ: valueText(lookup(node, "t")), handle: valueText(lookup(node, "h")), slots: r.slots(path)})
	}
	return s, notes, nil
}

// loadContracts reads the contracts of every component type in the load, in one registry call;
// a client without the registry channel, or a read that fails, leaves instance flags alone.
func (r *invReader) loadContracts(t *Tools, depth int) string {
	tl, ok := t.Ctx.Session.Client.(plan.TypeLoader)
	if !ok {
		return ""
	}
	specs, seen := []string{}, map[string]bool{}
	for _, path := range r.nodes.Keys() {
		if spec, _ := lookup(box.Node(r.nodes, path), "t").(string); spec != "" && !seen[spec] && strings.Count(path, "/") < depth {
			seen[spec] = true
			specs = append(specs, spec)
		}
	}
	ct := plan.NewContracts(tl)
	if err := ct.Load(specs); err != nil {
		return "the type contracts could not be read (" + plan.PyStr(err) + "): only slots flagged transient on " +
			"the instance are left out, so a slot its type declares transient can show as changed"
	}
	r.ct = ct
	return ""
}

// slots are the recorded slots of the component at path, in load order.
func (r *invReader) slots(path string) *pyjson.Object {
	out := pyjson.NewObject()
	node, kids := box.Node(r.nodes, path), box.Children(r.nodes, path)
	if r.chosen != nil {
		for _, name := range r.chosen {
			out.Set(name, r.chosenValue(path, name, box.Node(kids, name)))
		}
		return out
	}
	for _, name := range kids.Keys() {
		kid := box.Node(kids, name)
		v, ok := kid.Get("v")
		if !ok || strings.HasPrefix(valueText(lookup(kid, "t")), "baja:Status") || r.runtime(node, kid) {
			continue
		}
		out.Set(name, valueText(v))
	}
	return out
}

// chosenValue is a slot the operator named: its value text, a Status value as "value {status}",
// nil when the load holds no such slot (at its default, or none).
func (r *invReader) chosenValue(path, name string, kid *pyjson.Object) any {
	if kid == nil {
		return nil
	}
	if v, ok := kid.Get("v"); ok {
		return valueText(v)
	}
	full := childPath(path, name)
	if value, status, err := box.StatusValue(r.nodes, full); err == nil {
		return valueText(value) + " {" + valueText(status) + "}"
	}
	return nil
}

// runtime says whether kid, a slot of node, is flagged transient on the instance or in its contract.
func (r *invReader) runtime(node, kid *pyjson.Object) bool {
	if f, _ := lookup(kid, "f").(string); strings.Contains(f, runtimeFlag) {
		return true
	}
	if r.ct == nil {
		return false
	}
	typ, _ := lookup(node, "t").(string)
	slot, err := r.ct.Declared(typ, kid)
	if err != nil || slot == nil {
		return false
	}
	f, _ := lookup(slot, "f").(string)
	return strings.Contains(f, runtimeFlag)
}

// valueText is a loaded value as text: a string as is, anything else as Python's str().
func valueText(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if v == nil {
		return ""
	}
	return pyjson.Str(v)
}

// invSlotPath is the slot path of path, a load path below base: base's slot part joined with it.
func invSlotPath(base, path string) string {
	root := base
	if i := strings.Index(base, "slot:"); i >= 0 {
		root = base[i:]
	}
	if path == "" {
		return root
	}
	if strings.HasSuffix(root, "/") {
		return root + path
	}
	return root + "/" + path
}
