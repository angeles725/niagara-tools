package blackbox

// n4_reorder_slots (#1168): the complete new order of a component's dynamic children, sent as the
// station's ReorderOp, with its inverse (the previous order) and a read-back of the order the
// station holds. The fake station applies the op, so these tests read the order it ends up with.

import (
	"slices"
	"strings"
	"testing"
)

// reorderRig is a Folder holding A, B and C in that order.
func reorderRig(t *testing.T) *dRig {
	d := destructiveRig(t, nil)
	for _, n := range []string{"A", "B", "C"} {
		d.Add("3", n, "baja:Folder")
	}
	return d
}

func reorderArgs(order ...any) Args { return Args{"parent_ord": Folder, "order": order} }

func TestReorderSlotsReordersReadsBackAndRollsBack(t *testing.T) {
	d := reorderRig(t)
	args := reorderArgs("C", "A", "B")
	dry := d.Srv.Dry("n4_reorder_slots", args)
	if ops := planOf(dry)["ops"]; !same(ops, []any{map[string]any{"nm": "o", "h": "3", "o": "C;A;B"}}) {
		t.Errorf("ops = %v", ops)
	}
	if inv := planOf(dry)["inverse"]; !same(inv, []any{map[string]any{"nm": "o", "h": "3", "o": "A;B;C"}}) {
		t.Errorf("inverse = %v", inv)
	}
	if got := d.children(); !slices.Equal(got, []string{"A", "B", "C"}) {
		t.Fatalf("children after the dry run = %v", got)
	}
	done := d.Srv.OK("n4_reorder_slots", Execute(args, dry["confirmation_token"]))
	if done["verdict"] != "verified" {
		t.Errorf("done = %v", done)
	}
	if got := d.children(); !slices.Equal(got, []string{"C", "A", "B"}) {
		t.Fatalf("children after the reorder = %v", got)
	}
	if obs := objects(done["observed"]); len(obs) != 1 || !same(obs[0]["order"], []any{"C", "A", "B"}) {
		t.Errorf("observed = %v", done["observed"])
	}
	if back := d.rollback(done["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if got := d.children(); !slices.Equal(got, []string{"A", "B", "C"}) {
		t.Errorf("children after the rollback = %v", got)
	}
}

type reorderCase struct {
	name  string
	order []any
	want  string
}

// refuseAll runs each case as a dry run that is refused with its wording, and checks the station
// still holds its order.
func (d *dRig) refuseAll(t *testing.T, cases []reorderCase) {
	t.Helper()
	for _, tc := range cases {
		text := d.Srv.Err("n4_reorder_slots", reorderArgs(tc.order...))
		if !strings.Contains(text, tc.want) {
			t.Errorf("%s: reply %q lacks %q", tc.name, text, tc.want)
		}
		if got := d.children(); !slices.Equal(got, []string{"A", "B", "C"}) {
			t.Errorf("%s: children = %v", tc.name, got)
		}
	}
	if lines := d.Lines("journal.jsonl"); len(lines) != 0 {
		t.Errorf("journal = %v", lines)
	}
}

// A list that is not a permutation of the dynamic children is refused before the token: nothing
// is journaled and the station keeps its order.
func TestReorderSlotsRefusesAListThatIsNotAPermutation(t *testing.T) {
	d := reorderRig(t)
	d.refuseAll(t, []reorderCase{
		{"a missing child", []any{"A", "B"}, "missing: C"},
		{"an unknown name", []any{"A", "B", "C", "Z"}, "'Z' is not a dynamic child of"},
		{"a repeated name", []any{"A", "B", "B"}, "'B' is listed twice"},
		{"an empty list", []any{}, "order needs"},
		{"a malformed name", []any{"B", "C d"}, "must be a plain slot name"},
		{"the order it already has", []any{"A", "B", "C"}, "already in that order"},
	})
}

// A slot the parent's type declares is frozen: it cannot be listed, and the rest is the whole order.
func TestReorderSlotsRefusesAFrozenSlot(t *testing.T) {
	d := reorderRig(t)
	d.Station.SetRegistry(map[string]any{"baja:Folder": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "A", "v": map[string]any{"nm": "p", "t": "baja:Folder"}}}}})
	d.refuseAll(t, []reorderCase{
		{"a frozen slot in the list", []any{"A", "C", "B"}, "'A' is a frozen slot of"},
		{"only the frozen slot", []any{"A"}, "frozen"},
		{"a missing dynamic child", []any{"C"}, "missing: B"},
	})
	dry := d.Srv.Dry("n4_reorder_slots", reorderArgs("C", "B"))
	if ops := planOf(dry)["ops"]; !same(ops, []any{map[string]any{"nm": "o", "h": "3", "o": "C;B"}}) {
		t.Errorf("frozen A stays out of the order: ops = %v", ops)
	}
}

// The rollback read-back of a rename still reads the old name back (the shared rollback code
// gained the reorder branch).
func TestRollbackOfARenameStillVerifies(t *testing.T) {
	d := reorderRig(t)
	args := Args{"items": []any{map[string]any{"parent_ord": Folder, "name": "B", "new_name": "Bee"}}}
	done := d.Srv.RunWrite("n4_rename_component", args)
	if got := d.children(); !slices.Equal(got, []string{"A", "Bee", "C"}) {
		t.Fatalf("children after the rename = %v", got)
	}
	if back := d.rollback(done["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	if got := d.children(); !slices.Equal(got, []string{"A", "B", "C"}) {
		t.Errorf("children after the rollback = %v", got)
	}
}
