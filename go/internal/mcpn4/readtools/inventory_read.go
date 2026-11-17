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
// A component below a runtime property (inventory_slots.go), such as a history extension's
// lastRecord (BHistoryExt flags readonly|transient), is left out with everything below it.

// invReader is one snapshot read: the load, the contracts (nil: instance flags only) and the
// slots argument (nil: configured properties).
type invReader struct {
	nodes  *pyjson.Object
	ct     *plan.Contracts
	chosen []string
}

// readInventory loads base and returns its components down to depth, at most maxComponents.
func (t *Tools) readInventory(base string, depth, maxComponents int, chosen []string) (*invSnapshot, []string, error) {
	// two more levels: the properties of the deepest components, and below them the value and
	// status of a Status slot and the slots of a link (which input it drives)
	nodes, _, err := t.timedLoad(base, depth+2)
	if err != nil {
		return nil, nil, err
	}
	r := &invReader{nodes: nodes, chosen: chosen}
	notes := []string{}
	if note := r.loadContracts(t, depth); note != "" {
		notes = append(notes, t.Ctx.Scrub(note))
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
		if path != "" && r.runtime(box.Node(nodes, parentPath(path)), node) {
			continue // a struct the station keeps at runtime, and everything below it
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
			"the instance, Status outs and linked inputs are left out, so a slot its type declares transient " +
			"(such as a history extension's lastRecord) can show as changed"
	}
	r.ct = ct
	return ""
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
