package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// SetTags is n4_set_tags: direct tags added (a metadata slot, flag m), changed (same type) or
// removed on 1 to 50 components, after the installed dictionaries validate them.
var SetTags = plan.Impl{ScopeOrds: tagOrdsArg, Plan: setTagsPlan, Readback: setTagsReadback}

// tagExpect is what the read-back checks on one target: the slot and body each add sent, and the
// slots each remove deleted.
type tagExpect struct {
	ord     string
	added   []tagAdd
	removed []tags.ID
}

func setTagsPlan(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
	ords, err := tagTargets(args)
	if err != nil {
		return nil, err
	}
	removes, err := tagRemoves(args)
	if err != nil {
		return nil, err
	}
	var adds []tagAdd
	if v, _ := args.Get("add"); len(asList(v)) > 0 {
		cat, err := tags.LoadCatalog(c)
		if err != nil {
			return nil, err
		}
		if adds, err = tagAdds(args, cat); err != nil {
			return nil, err
		}
	}
	if n := (len(adds) + len(removes)) * len(ords); n == 0 {
		return nil, plan.ToolError("nothing to change: give add and/or remove")
	} else if n > MaxTagOps {
		return nil, plan.ToolError(fmt.Sprintf("the batch would send %d ops (targets x tags); at most %d: split it", n, MaxTagOps))
	}
	p := &plan.Planned{Ops: []any{}, Notes: []any{}, Targets: pyjson.NewObject()}
	var inverse, expects []any
	for _, o := range ords {
		h, nodes, err := plan.Handle(c, o, 1)
		if err != nil {
			return nil, err
		}
		p.Targets.Set(o, h)
		exp := tagExpect{ord: o}
		for _, a := range adds {
			op, inv, note, err := tagAddOp(h, nodes, a, o)
			if err != nil {
				return nil, err
			}
			if note != "" {
				p.Notes = append(p.Notes, note)
			}
			if op != nil {
				p.Ops, inverse = append(p.Ops, op), append(inverse, inv)
				exp.added = append(exp.added, a)
			}
		}
		for _, id := range removes {
			op, inv, err := tagRemoveOp(h, nodes, id, o)
			if err != nil {
				return nil, err
			}
			p.Ops, inverse = append(p.Ops, op), append(inverse, inv)
			exp.removed = append(exp.removed, id)
		}
		expects = append(expects, exp)
	}
	if len(p.Ops) == 0 {
		return nil, plan.ToolError("nothing to change: every tag already holds the requested value")
	}
	for i := len(inverse) - 1; i >= 0; i-- { // undone last first
		p.Inverse = append(p.Inverse, inverse[i])
	}
	p.Data = expects
	return p, nil
}

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

func asList(v any) []any { l, _ := v.([]any); return l }
