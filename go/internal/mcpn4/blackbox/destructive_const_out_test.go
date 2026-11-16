package blackbox

// The remove snapshot carries a constant block's configured out, so the rollback of the remove
// re-creates it with its value and its read-back compares it.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestRollbackOfARemovedNumericConstRestoresItsConfiguredOut(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	k := d.AddWithOut("K", 10.0)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(k.Name))
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if got := d.childValues(k.Name, "out"); got["value"] != "10.0" {
		t.Errorf("K.out = %v, want value 10.0", got)
	}
}

func TestRollbackOfARemovedNumericConstThatLosesItsOutIsAMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	k := d.AddWithOut("K", 10.0)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(k.Name))
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) { removeChild(n, "out") })
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "mismatch" {
		t.Errorf("back = %v", back)
	}
}

func TestTheRemoveSnapshotStillDropsAComputedOut(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	w := d.Add("3", "W", "control:NumericWritable")
	d.Set(w.Handle, "out", StatusNumeric(10.0, "0"))
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(w.Name))
	sent := d.f.record()
	d.rollback(removed["batch_id"])
	op := addOf(sent(), w.Name)
	if op == nil || destructiveContainsAny(field(Dig(op, "b", "s"), "n"), "out") {
		t.Errorf("W add = %v", op)
	}
}
