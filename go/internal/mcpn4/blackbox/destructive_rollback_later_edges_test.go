package blackbox

// Edge probes of the create-rollback later-work guard, from the independent verifier of delta 1.

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func journalHash(t *testing.T, d *dRig) string {
	t.Helper()
	var all []string
	for _, l := range d.Lines("journal.jsonl") {
		all = append(all, fmt.Sprint(l))
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(all, "\n"))))
}

// later batch rolled back, then re-done: the re-do is later work.
func TestRollbackLaterWorkEdgeRolledBackThenRedone(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	b1 := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	d.rollback(b1["batch_id"])
	b2 := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	h := journalHash(t, d)
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, b2["batch_id"].(string)+" (n4_create_component)")
	if strings.Contains(text, b1["batch_id"].(string)) {
		t.Errorf("the rolled-back batch is listed: %q", text)
	}
	if journalHash(t, d) != h {
		t.Errorf("journal changed on refusal")
	}
}

// sibling with a common prefix only: no refusal.
func TestRollbackLaterWorkEdgeSiblingPrefixOnly(t *testing.T) {
	d := destructiveRig(t, nil)
	out, _ := d.createFolder("Box")
	_, box2 := d.createFolder("Box2")
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box2, "name": "Calc", "type": "kitControl:NumericConst"})
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": "BoxCalc", "type": "kitControl:NumericConst"})
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
}

// nested: create Box, Box/Sub, Box/Sub/Calc; Sub's rollback names Calc's batch only; unwinding newest first works.
func TestRollbackLaterWorkEdgeNestedChainUnwinds(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	sub := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Sub", "type": "baja:Folder"})
	calc := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box + "/Sub", "name": "Calc", "type": "kitControl:NumericConst"})
	text := d.refusedRollback(sub["batch_id"])
	wantAll(t, text, calc["batch_id"].(string))
	if strings.Contains(text, sub["batch_id"].(string)+" (") {
		t.Errorf("the batch itself is listed as later work: %q", text)
	}
	text = d.refusedRollback(out["batch_id"])
	wantAll(t, text, calc["batch_id"].(string), sub["batch_id"].(string))
	for _, id := range []any{calc["batch_id"], sub["batch_id"], out["batch_id"]} {
		if back := d.rollback(id); back["verdict"] != "verified" {
			t.Fatalf("rollback %v = %v", id, back)
		}
	}
	if slices.Contains(d.children(), "Box") {
		t.Errorf("Box still there")
	}
}

// copy_component into the created folder.
func TestRollbackLaterWorkEdgeCopyIntoCreated(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	d.Add(d.node("Box").Handle, "Seed", "") // hand-made, so also an extra
	cp := d.Srv.RunWrite("n4_copy_component", Args{"parent_ord": box, "name": "Seed", "new_names": []any{"Seed2"}})
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, cp["batch_id"].(string)+" (n4_copy_component)", "Seed", "Seed2")
}

// copying the created component itself (the copy lands beside it): not under it, no refusal.
func TestRollbackLaterWorkEdgeCopyOfCreatedIsBeside(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	d.Srv.RunWrite("n4_copy_component", Args{"parent_ord": Folder, "name": "Pump", "new_names": []any{"Pump2"}})
	r := d.Srv.Call("n4_rollback", Args{"batch_id": out["batch_id"]})
	if r.IsError {
		t.Errorf("refused: %s", r.Text)
	}
}

// journaled link whose target is inside the created component.
func TestRollbackLaterWorkEdgeJournaledLinkTargetInside(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	src, _ := d.addOut("Src", "", num(1.0))
	ln := d.Srv.RunWrite("n4_create_link", Args{"source_ord": Folder + "/" + src, "source_slot": "out",
		"target_ord": Folder + "/Pump", "target_slot": "in10"})
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, ln["batch_id"].(string)+" (n4_create_link)")
}

// a token from a dry run taken before the later work: the execution must not remove it.
func TestRollbackLaterWorkEdgeStaleTokenAfterLaterWork(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	dry := d.Srv.Dry("n4_rollback", Args{"batch_id": out["batch_id"]})
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	r := d.Srv.Call("n4_rollback", Execute(Args{"batch_id": out["batch_id"]}, dry["confirmation_token"]))
	if !r.IsError || d.node("Box", "Calc") == nil {
		t.Errorf("stale token executed: %v; Box/Calc present=%v", r.Text, d.node("Box", "Calc") != nil)
	}
}

// a hand-added grandchild (two levels below the created component).
