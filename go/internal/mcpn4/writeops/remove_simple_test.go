package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// simpleSlotLoads is a station holding point P with a metadata Double slot (a direct tag) and a
// relation struct. Like the lab station (4.15), a load of a non-component slot's ORD answers the
// component that holds it: there is no load event for a simple value.
func simpleSlotLoads() *loads {
	tag := obj("nm", "p", "n", "tst$3acap", "f", "m", "t", "baja:Double", "v", "4.5")
	relID := obj("nm", "p", "n", "relationId", "v", "tst:feeds")
	relSrc := obj("nm", "p", "n", "sourceOrd", "v", "h:5")
	rel := obj("nm", "p", "n", "r", "t", "baja:Relation", "s", []any{relID, relSrc})
	root := obj("nm", "p", "h", "6", "t", "control:NumericWritable", "s", []any{tag, rel})
	tree := obj("", root, "tst$3acap", tag, "r", rel, "r/relationId", relID, "r/sourceOrd", relSrc)
	return &loads{nodes: obj("station:|slot:/P", tree, "station:|slot:/P/tst$3acap", tree,
		"station:|slot:/P/r", tree)}
}

// TestRemoveSimpleSlotInverseReaddsThatSlot: the inverse of removing a simple slot re-adds that
// slot (its type, value and flags), never the component that holds it.
func TestRemoveSimpleSlotInverseReaddsThatSlot(t *testing.T) {
	p, err := writeops.RemoveComponent.Plan(simpleSlotLoads(), obj("parent_ord", "station:|slot:/P", "name", "tst$3acap"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"nm": "a", "h": "6", "n": "tst$3acap", "b": {"nm": "p", "t": "baja:Double", "v": "4.5"}, "f": "m"}]`
	if got := marshal(t, p.Inverse); got != want {
		t.Fatalf("inverse\n got %s\nwant %s", got, want)
	}
	if got := marshal(t, p.Ops); got != `[{"nm": "v", "h": "6", "n": "tst$3acap"}]` {
		t.Fatalf("ops %s", got)
	}
}

// TestRemoveStructSlotInverseKeepsItsFields: a struct slot (a relation) is re-added with its fields.
func TestRemoveStructSlotInverseKeepsItsFields(t *testing.T) {
	p, err := writeops.RemoveComponent.Plan(simpleSlotLoads(), obj("parent_ord", "station:|slot:/P", "name", "r"))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"nm": "a", "h": "6", "n": "r", "b": {"nm": "p", "t": "baja:Relation", "s": [` +
		`{"nm": "p", "n": "relationId", "v": "tst:feeds"}, {"nm": "p", "n": "sourceOrd", "v": "h:5"}]}}]`
	if got := marshal(t, p.Inverse); got != want {
		t.Fatalf("inverse\n got %s\nwant %s", got, want)
	}
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
