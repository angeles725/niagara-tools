package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// childNodes is the flat load of child nm of the parent (handle parentH), whether nm is a plain
// slot, and for a plain slot the load of the component that holds it. A simple or struct slot (a
// tag, a relation) has no load of its own: the station answers its ORD with the component that
// holds it (lab 2026-11-15, 4.15: the inverse re-added the parent under the slot's name), so when
// the child's load comes back with the parent's handle its subtree is re-keyed from that load
// (taken to the snapshot depth, so a struct keeps its fields).
func childNodes(c plan.Client, parentH any, nm, childOrd string) (nodes, holder *pyjson.Object, plain bool, err error) {
	h, holder, err := plan.Handle(c, childOrd, SnapshotDepth+1)
	if err != nil || !sameHandle(h, parentH) {
		return holder, nil, false, err
	}
	nodes = pyjson.NewObject()
	for _, p := range holder.Keys() {
		v, _ := holder.Get(p)
		if p == nm {
			nodes.Set("", v)
		} else if rest, ok := strings.CutPrefix(p, nm+"/"); ok {
			nodes.Set(rest, v)
		}
	}
	return nodes, holder, true, nil
}

// plainSlotLinks prepares the link checks of a removed plain slot nm: the links it sources are
// found by the scan as links from the holder (parentH, path "") read from nm only (linkScan's
// slot filter), and each link of the holder that writes nm is recorded as an unlinked_input, as
// a component's linked input is: the remove drops it and a rollback does not re-create it.
func plainSlotLinks(holder, body *pyjson.Object, nm string, parentH any, handles map[string]string,
	inverse, notes []any) ([]any, []any, bool, error) {
	key, err := pyjson.DictKey(parentH)
	if err != nil {
		return nil, nil, false, unhashable(parentH)
	}
	linked := false
	handles[key] = ""
	for _, child := range kidsOf(box.Node(holder, ""), 1) {
		if !isLink(get(child, "t")) {
			continue
		}
		slots := pyjson.NewObject()
		for _, k := range kidsOf(child, 1) {
			slots.Set(name(k), get(k, "v"))
		}
		if get(slots, "targetSlotName") != nm {
			continue
		}
		// its last value is the snapshot body, as a component's linked input keeps its own (prior)
		spec := obj("source_slot", get(slots, "sourceSlotName"), "target_path", "", "target_slot", nm, "prior", body)
		notes = append(notes, "link into "+pyjson.StrRepr(nm)+" ("+nm+" <- "+pyjson.Str(get(slots, "sourceSlotName"))+
			") comes from outside the removed subtree: not restored; its last value is recorded and reported by n4_rollback")
		inverse, linked = append(inverse, obj("unlinked_input", spec)), true
	}
	return inverse, notes, linked, nil
}

// fromSlot keeps the scanned links whose source slot is slot ("" keeps them all): a plain slot's
// scan matches every link from its holder, and only those it sources break with it.
func fromSlot(found []any, slot string) []any {
	if slot == "" {
		return found
	}
	out := []any{}
	for _, f := range found {
		if get(f.(*pyjson.Object), "source_slot") == slot {
			out = append(out, f)
		}
	}
	return out
}

// slotAdd is the re-add op of the removed child: a plain slot keeps its flags (a tag's metadata
// flag), as the station's AddOp carries them ("f").
func slotAdd(parentH any, nm string, body, root *pyjson.Object, plain bool) *pyjson.Object {
	op := obj("nm", "a", "h", parentH, "n", nm, "b", body)
	if f, _ := get(root, "f").(string); plain && f != "" {
		op.Set("f", f)
	}
	return op
}

// sameHandle is a == b for two handles as dict keys (a string, or an unhashable value: never equal).
func sameHandle(a, b any) bool {
	ka, errA := pyjson.DictKey(a)
	kb, errB := pyjson.DictKey(b)
	return errA == nil && errB == nil && ka == kb
}
