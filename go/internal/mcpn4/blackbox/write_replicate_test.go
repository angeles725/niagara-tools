package blackbox

// n4_replicate_points copies M points of one parent onto N target parents in one batch under one
// confirmation token (field retro 2026-11-16, row U10: a client B field job spent 630 calls and
// 39 minutes on 112 points, one token per copy, and ran out of budget). Each add is one write of
// --max-writes: the dry run shows the count and is refused up front when it exceeds what remains.

import (
	"strings"
	"testing"
)

// replicaRig is a Folder holding the source points K1 and K2 and the empty targets A and B.
func replicaRig(t *testing.T, maxWrites int) (*dRig, Component) {
	d := destructiveRig(t, func(r *Rig, o *Options) { o.MaxWrites = maxWrites })
	d.withContracts()
	d.AddWithOut("K1", 1.0)
	d.AddWithOut("K2", 2.0)
	a := d.Add("3", "A", "baja:Folder")
	d.Add("3", "B", "baja:Folder")
	return d, a
}

func replicateArgs() Args {
	return Args{"source_parent_ord": Folder, "points": []any{"K1", "K2"}, "targets": []any{
		map[string]any{"parent_ord": Folder + "/A"}, map[string]any{"parent_ord": Folder + "/B"}}}
}

func TestOneTokenReplicatesEveryPointOntoEveryTarget(t *testing.T) {
	d, _ := replicaRig(t, 200)
	dry := d.Srv.Dry("n4_replicate_points", replicateArgs())
	if ops := objects(planOf(dry)["ops"]); len(ops) != 4 {
		t.Fatalf("plan has %d ops, want 4 (2 points x 2 targets): %v", len(ops), planOf(dry)["ops"])
	}
	if !strings.Contains(jsonText(planOf(dry)["notes"]), "4 write(s) of the budget") {
		t.Errorf("notes do not show the write count: %v", planOf(dry)["notes"])
	}
	done := d.Srv.OK("n4_replicate_points", Execute(replicateArgs(), dry["confirmation_token"]))
	if done["verdict"] != "verified" {
		t.Errorf("verdict = %v (%v)", done["verdict"], done)
	}
	for _, target := range []string{"A", "B"} {
		for k, want := range map[string]string{"K1": "1.0", "K2": "2.0"} {
			if got := d.childValues(target, k, "out"); got["value"] != want {
				t.Errorf("%s/%s.out = %v, want value %s", target, k, got, want)
			}
		}
	}
	budget, _ := done["write_budget"].(map[string]any)
	if budget["executed"] != float64(4) || budget["remaining"] != float64(196) {
		t.Errorf("write_budget = %v, want 4 executed (one per add)", budget)
	}
	if inv := objects(done["inverse"]); len(inv) != 4 {
		t.Errorf("inverse = %v, want one remove per copy", done["inverse"])
	}
}

func TestAReplicationLargerThanTheRemainingBudgetIsRefusedUpFront(t *testing.T) {
	d, _ := replicaRig(t, 3)
	before := d.childNames(d.node("A"))
	text := d.Srv.Err("n4_replicate_points", replicateArgs())
	if !strings.Contains(text, "write budget too small") || !strings.Contains(text, "needs 4 write(s)") ||
		!strings.Contains(text, "only 3 remain") {
		t.Errorf("refusal = %s", text)
	}
	if got := d.childNames(d.node("A")); len(got) != len(before) {
		t.Errorf("A changed: %v", got)
	}
}

func TestAReplicaThatWouldShadowAnExistingChildIsRefused(t *testing.T) {
	d, a := replicaRig(t, 200)
	d.Add(a.Handle, "K2", "")
	text := d.Srv.Err("n4_replicate_points", replicateArgs())
	if !strings.Contains(text, "'K2' already exists under "+Folder+"/A") {
		t.Errorf("refusal = %s", text)
	}
}
