package writeops

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// childNodes is the flat load of child nm of the parent (handle parentH) and whether nm is a plain
// slot. A simple or struct slot (a tag, a relation) has no load of its own: the station answers
// its ORD with the component that holds it (lab 2026-11-15, 4.15: the inverse re-added the
// parent under the slot's name), so when the child's load comes back with the parent's handle
// its subtree is taken from the parent's load instead.
func childNodes(c plan.Client, parentH any, parentNodes *pyjson.Object, nm, childOrd string) (*pyjson.Object, bool, error) {
	h, nodes, err := plan.Handle(c, childOrd, SnapshotDepth+1)
	if err != nil || !sameHandle(h, parentH) {
		return nodes, false, err
	}
	nodes = pyjson.NewObject()
	for _, p := range parentNodes.Keys() {
		v, _ := parentNodes.Get(p)
		if p == nm {
			nodes.Set("", v)
		} else if rest, ok := strings.CutPrefix(p, nm+"/"); ok {
			nodes.Set(rest, v)
		}
	}
	return nodes, true, nil
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
