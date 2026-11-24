package blackbox

// n4_remove_component compact=true (issue #930): the dry run summarises the snapshot instead of listing it.

import (
	"slices"
	"testing"
)

func TestCompactDryRunSummarisesTheSnapshotAndKeepsTheHash(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	full := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	args := removeArgs(nn)
	args["compact"] = true
	dry := d.Srv.Dry("n4_remove_component", args)
	plan := planOf(dry)
	if _, ok := plan["inverse"]; ok {
		t.Errorf("compact plan still lists inverse: %v", plan["inverse"])
	}
	want := map[string]any{"components": float64(3), "links_inside": float64(1), "unlinked_inputs": float64(0),
		"outgoing_links": float64(0), "inverse_ops": float64(2)}
	if !same(plan["inverse_summary"], want) {
		t.Errorf("inverse_summary = %v, want %v", plan["inverse_summary"], want)
	}
	if dry["plan_hash"] != full["plan_hash"] {
		t.Errorf("plan_hash %v differs from the full dry run's %v", dry["plan_hash"], full["plan_hash"])
	}
	if !same(plan["ops"], planOf(full)["ops"]) || !slices.Contains(d.children(), nn) {
		t.Errorf("ops = %v, children %v", plan["ops"], d.children())
	}
}

func TestCompactRemoveExecutesAndRollsBackLikeTheFullOne(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	args := removeArgs(nn)
	args["compact"] = true
	removed := d.Srv.RunWrite("n4_remove_component", args)
	if removed["verdict"] != "verified" || slices.Contains(d.children(), nn) {
		t.Fatalf("removed = %v, children %v", removed, d.children())
	}
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "verified" || !slices.Contains(d.children(), nn) {
		t.Errorf("back = %v, children %v", back, d.children())
	}
}

func TestRemoveWithoutCompactKeepsItsInverse(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))
	if _, ok := plan["inverse_summary"]; ok || len(objects(plan["inverse"])) != 2 {
		t.Errorf("plan = %v", plan)
	}
}
