package readtools

// #890 (retro R5 ΔD1): a frozen property written back to its declared default while a link
// targets it (a BooleanWritable's in16 fed by a schedule) comes back from the station by name
// only: no type, value or children, at most instance flags. It is at its declared default, so
// n4_read_slots lists that default (default: true) instead of type and value null, and
// plan.Resolve hands set_slot's read-back and n4_rollback the declared node instead of a node
// box.StatusValue refuses ("is not a StatusBoolean/StatusNumeric").

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestASlotListedByNameOnlyIsListedOnceAtItsDeclaredDefault(t *testing.T) {
	for _, node := range []*pyjson.Object{obj("nm", "p", "n", "in16"), obj("nm", "p", "n", "in16", "f", "s")} {
		list := bwList(t, "in16", node)
		if len(list) != 18 {
			t.Errorf("%s: %d slots, want 18 (in16 once)", pyjson.Repr(node), len(list))
		}
		e := bwSlots(t, "in16", node)["in16"]
		for k, v := range map[string]any{"type": "baja:StatusBoolean", "value": "false", "status": "40", "default": true} {
			if g, _ := e.Get(k); g != v {
				t.Errorf("%s: in16 %s = %v, want %v", pyjson.Repr(node), k, g, v)
			}
		}
	}
}

func TestResolveReplacesASlotListedByNameOnlyWithItsDeclaredDefault(t *testing.T) {
	f := &ctfLoader{answer: ctfAnswer(bwContract())}
	nodes := obj("", obj("t", "control:BooleanWritable"), "in16", obj("nm", "p", "n", "in16"))
	got, err := plan.Resolve(f, nodes, "in16")
	if err != nil {
		t.Fatal(err)
	}
	value, status, err := box.StatusValue(got, "in16")
	if err != nil || value != "false" || status != "40" {
		t.Fatalf("value %v, status %v, err %v", value, status, err)
	}
	if n := box.Node(nodes, "in16"); pyjson.Repr(n) != pyjson.Repr(obj("nm", "p", "n", "in16")) {
		t.Errorf("the loaded nodes changed: %s", pyjson.Repr(n))
	}
	listed := obj("", obj("t", "control:BooleanWritable"), "in16", obj("n", "in16", "t", "baja:StatusBoolean"))
	if same, _ := plan.Resolve(f, listed, "in16"); same != listed {
		t.Error("a listed slot is no longer left as loaded")
	}
}
