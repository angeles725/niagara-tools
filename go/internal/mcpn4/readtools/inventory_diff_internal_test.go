package readtools

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// comp is a snapshot component whose slots are name, value pairs (a nil value: not sent).
func comp(path, typ, handle string, slots ...any) invComponent {
	return invComponent{path: path, typ: typ, handle: handle, slots: obj(slots...)}
}

func asJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDiffInventoryListsEveryKindOfChangeByPath(t *testing.T) {
	a := &invSnapshot{name: "a", components: []invComponent{
		comp("slot:/F", "baja:Folder", "3"),
		comp("slot:/F/Pump", "kitControl:NumericConst", "10", "facets", "units=C", "gone", "1"),
		comp("slot:/F/Old", "kitControl:NumericConst", "11"),
		comp("slot:/F/Re", "baja:Folder", "12"),
		comp("slot:/F/Moved", "baja:Folder", "13"),
	}}
	b := &invSnapshot{name: "b", components: []invComponent{
		comp("slot:/F", "baja:Folder", "3"),
		comp("slot:/F/Pump", "kitControl:NumericConst", "10", "facets", "units=F", "fresh", "2"),
		comp("slot:/F/Re", "kitControl:NumericConst", "12", "facets", "x"),
		comp("slot:/F/Moved", "baja:Folder", "20"),
		comp("slot:/F/New", "baja:Folder", "21"),
	}}
	got := diffInventory(a, b).reply(a, b, 10)
	for key, want := range map[string]string{
		"added":   `[{"path": "slot:/F/New", "type": "baja:Folder"}]`,
		"removed": `[{"path": "slot:/F/Old", "type": "kitControl:NumericConst"}]`,
		"changed": `[{"path": "slot:/F/Pump", "slot": "facets", "before": "units=C", "after": "units=F"}, ` +
			`{"path": "slot:/F/Pump", "slot": "gone", "before": "1", "after": null}, ` +
			`{"path": "slot:/F/Pump", "slot": "fresh", "before": null, "after": "2"}]`,
		"type_changed":   `[{"path": "slot:/F/Re", "before": "baja:Folder", "after": "kitControl:NumericConst"}]`,
		"handle_changed": `[{"path": "slot:/F/Moved", "before": "13", "after": "20"}]`,
		"counts":         `{"added": 1, "removed": 1, "changed": 3, "type_changed": 1, "handle_changed": 1, "unchanged": 2}`,
		"truncated":      `false`,
		"notes":          `[]`,
	} {
		if s := asJSON(t, lookup(got, key)); s != want {
			t.Errorf("%s = %s, want %s", key, s, want)
		}
	}
}

func TestDiffInventoryOfTwoEqualSnapshotsIsEmpty(t *testing.T) {
	a := &invSnapshot{name: "a", components: []invComponent{comp("slot:/F/P", "x:T", "1", "v", "1", "d", nil)}}
	got := diffInventory(a, a).reply(a, a, 10)
	if s := asJSON(t, lookup(got, "counts")); s != `{"added": 0, "removed": 0, "changed": 0, "type_changed": 0, "handle_changed": 0, "unchanged": 1}` {
		t.Errorf("counts %s", s)
	}
}

func TestDiffInventoryCapsEachListAtLimitAndCountsAll(t *testing.T) {
	a := &invSnapshot{name: "a"}
	b := &invSnapshot{name: "b", components: []invComponent{comp("/1", "t", "1"), comp("/2", "t", "2"), comp("/3", "t", "3")}}
	got := diffInventory(a, b).reply(a, b, 2)
	if s := asJSON(t, lookup(got, "added")); s != `[{"path": "/1", "type": "t"}, {"path": "/2", "type": "t"}]` {
		t.Errorf("added %s", s)
	}
	if lookup(got, "truncated") != true || asJSON(t, lookup(lookup(got, "counts").(*pyjson.Object), "added")) != "3" {
		t.Errorf("truncated %v counts %s", lookup(got, "truncated"), asJSON(t, lookup(got, "counts")))
	}
}

func TestInventoryNotesNameEveryMismatch(t *testing.T) {
	a := &invSnapshot{station: "S1", base: "station:|slot:/A", depth: 2, contracts: true}
	b := &invSnapshot{station: "S2", base: "station:|slot:/B", depth: 2, truncated: true, chosen: []string{"facets"}}
	notes := strings.Join(inventoryNotes(a, b), "\n")
	for _, want := range []string{"different stations (S1, S2)", "different subtrees", "different slots",
		"max_components", "without the type contracts"} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes %q lack %q", notes, want)
		}
	}
}

func TestInvSlotPathJoinsTheBaseSlotPart(t *testing.T) {
	for _, c := range [][3]string{{"station:|slot:/", "", "slot:/"}, {"station:|slot:/", "F/P", "slot:/F/P"},
		{"station:|slot:/F", "P", "slot:/F/P"}, {"station:|slot:/F", "", "slot:/F"}} {
		if got := invSlotPath(c[0], c[1]); got != c[2] {
			t.Errorf("invSlotPath(%q, %q) = %q, want %q", c[0], c[1], got, c[2])
		}
	}
}
