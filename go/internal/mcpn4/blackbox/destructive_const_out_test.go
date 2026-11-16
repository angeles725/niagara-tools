package blackbox

// The remove snapshot carries a constant block's configured out, so the rollback of the remove
// re-creates it with its value and its read-back compares it.

import (
	"encoding/json"
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

// A remove planned on a station whose registry channel answers an error keeps the snapshot it took
// before contracts were read (out left out), instead of failing a remove that worked before.
func TestARemoveWhoseRegistryCallFailsKeepsTheBaselineSnapshot(t *testing.T) {
	d := destructiveRig(t, nil)
	k := d.AddWithOut("K", 10.0)
	d.f.set(func(f *faults) {
		f.raw = func(frame map[string]any) (int, []byte) {
			for _, m := range objects(frame["m"]) {
				if m["c"] == "reg" {
					data, _ := json.Marshal(map[string]any{"v": "2.3", "p": "box", "n": frame["n"], "m": []any{
						map[string]any{"c": "reg", "k": "loadTypes", "r": 0, "t": "e", "b": map[string]any{"m": "registry denied"}}}})
					return 200, data
				}
			}
			return 0, nil
		}
	})
	if out := d.Srv.RunWrite("n4_remove_component", removeArgs(k.Name)); out["verdict"] != "verified" {
		t.Errorf("remove = %v", out)
	}
}
