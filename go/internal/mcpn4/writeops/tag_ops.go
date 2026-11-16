package writeops

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// tagAddOp is the op and inverse of one add on one target: a new metadata slot (inverse: remove
// it), or a set of the tag's value when it holds another (inverse: the value it held). An
// existing tag of another type, or a slot of that name that is no tag, is refused.
func tagAddOp(h any, nodes *pyjson.Object, a tagAdd, o string) (op, inv *pyjson.Object, note string, err error) {
	slot := a.id.Slot()
	node := box.Node(nodes, slot)
	if node == nil {
		return obj("nm", "a", "h", h, "n", slot, "b", a.body, "f", "m"), obj("nm", "v", "h", h, "n", slot), "", nil
	}
	if !isTag(node, slot) {
		return nil, nil, "", plan.ToolError(o + " has a slot " + slot + " that is not a direct tag: refusing")
	}
	if t := get(node, "t"); t != get(a.body, "t") {
		return nil, nil, "", plan.ToolError(a.id.String() + " on " + o + " is a " + pyjson.Str(t) + " tag; changing a tag's " +
			"type is refused: remove it, then add it")
	}
	if box.PyEqual(get(node, "v"), get(a.body, "v")) {
		return nil, nil, a.id.String() + " on " + o + " already holds that value: no op", nil
	}
	prior := obj("nm", "p", "t", get(node, "t"), "v", get(node, "v"))
	return obj("nm", "s", "h", h, "n", slot, "b", a.body), obj("nm", "s", "h", h, "n", slot, "b", prior), "", nil
}

// tagRemoveOp is the remove of one direct tag; its inverse re-adds it with its type, value and
// flags.
func tagRemoveOp(h any, nodes *pyjson.Object, id tags.ID, o string) (op, inv *pyjson.Object, err error) {
	slot := id.Slot()
	node := box.Node(nodes, slot)
	if node == nil || !isTag(node, slot) {
		return nil, nil, plan.ToolError(o + " has no direct tag " + id.String() + " (n4_read_tags lists its tags)")
	}
	body := obj("nm", "p", "t", get(node, "t"), "v", get(node, "v"))
	return obj("nm", "v", "h", h, "n", slot), obj("nm", "a", "h", h, "n", slot, "b", body, "f", get(node, "f")), nil
}

// isTag is whether a loaded slot is a direct tag (tags.IsTagSlot).
func isTag(node *pyjson.Object, slot string) bool {
	nm, _ := get(node, "nm").(string)
	flags, _ := get(node, "f").(string)
	return tags.IsTagSlot(nm, slot, flags, pyjson.Truthy(get(node, "h")))
}
