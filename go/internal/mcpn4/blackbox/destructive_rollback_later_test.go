package blackbox

// Rollback of a create never silently removes later work (bench retro 2026-11-15, delta 1):
// the rollback of a batch that created a component is refused when a later batch worked under
// it (journal) or when it now holds children or links its create did not make (live).

import (
	"slices"
	"strings"
	"testing"
)

// createFolder is n4_create_component of a baja:Folder under the Folder with a wire-sheet
// position; it returns the batch reply and the created folder's ORD.
func (d *dRig) createFolder(name string) (map[string]any, string) {
	d.T.Helper()
	out := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": name, "type": "baja:Folder",
		"wire_sheet": Args{"x": 1, "y": 2}})
	return out, Folder + "/" + name
}

// rollbackBatches counts the journaled batches that are rollbacks.
func (d *dRig) rollbackBatches() int {
	n := 0
	for _, l := range d.Lines("journal.jsonl") {
		if l["phase"] == "intent" && l["rollback_of"] != nil {
			n++
		}
	}
	return n
}

// refusedRollback runs n4_rollback of id expecting a refusal, and checks the refusal changed
// nothing: the Folder's children are the same and no rollback batch was journaled.
func (d *dRig) refusedRollback(id any) string {
	d.T.Helper()
	before, rollbacks := d.children(), d.rollbackBatches()
	text := d.Srv.Err("n4_rollback", Args{"batch_id": id})
	if after := d.children(); !slices.Equal(after, before) {
		d.T.Errorf("children %v -> %v after a refused rollback", before, after)
	}
	if n := d.rollbackBatches(); n != rollbacks {
		d.T.Errorf("rollback batches %d -> %d after a refused rollback", rollbacks, n)
	}
	return text
}

func wantAll(t *testing.T, text string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(text, p) {
			t.Errorf("refusal %q lacks %q", text, p)
		}
	}
}

// The bench case: later batches added a block under the created folder and linked into it.
func TestRollbackOfACreateRefusesWhenLaterBatchesWorkedUnderIt(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	block := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	src, _ := d.addOut("Src", "", num(1.0))
	link := d.Srv.RunWrite("n4_create_link", Args{"source_ord": Folder + "/" + src, "source_slot": "out",
		"target_ord": box + "/Calc", "target_slot": "in10"})
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, box, block["batch_id"].(string)+" (n4_create_component)",
		link["batch_id"].(string)+" (n4_create_link)", "newest first")
	if i, j := strings.Index(text, link["batch_id"].(string)), strings.Index(text, block["batch_id"].(string)); i > j {
		t.Errorf("later batches are not listed newest first: %q", text)
	}
	if d.node("Box", "Calc") == nil {
		t.Errorf("the block under Box was removed")
	}
}

// Journal only: a later set-slot on the created block leaves no new child, the journal still knows.
func TestRollbackOfACreateRefusesWhenALaterBatchSetASlotOnIt(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	d.Set(d.node("Pump").Handle, "out", StatusNumeric(1.0, "0")) // the fake omits a default out
	set := d.Srv.RunWrite("n4_set_slot", setOut("Pump"))
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, Folder+"/Pump", set["batch_id"].(string)+" (n4_set_slot)")
}

// A later batch rolled back itself is no later work: the create rolls back as before.
func TestRollbackOfACreateIgnoresALaterBatchAlreadyRolledBack(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	block := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	if back := d.rollback(block["batch_id"]); back["verdict"] != "verified" {
		t.Fatalf("rollback of the block = %v", back)
	}
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" || slices.Contains(d.children(), "Box") {
		t.Errorf("back = %v, children %v", back, d.children())
	}
}

// Live only: a child and a link added by hand (Workbench), with no journal entry.
func TestRollbackOfACreateRefusesWhenItHoldsAChildNoBatchMade(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	d.Add(d.node("Box").Handle, "Manual", "")
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, box, "Manual")
}

func TestRollbackOfACreateRefusesWhenALinkWasAddedToItByHand(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	src := d.AddWithOut("Src", 1.0)
	name := d.Link(src, "out", Component{"Pump", d.node("Pump").Handle}, "in10")
	text := d.refusedRollback(out["batch_id"])
	wantAll(t, text, Folder+"/Pump", name)
}

// Nothing added under the created folder (its wsAnnotation is the create's own slot), and a later
// batch under a sibling whose name only starts with it (/Box does not cover /Box2): unchanged.
func TestRollbackOfACreateWithNothingAddedStillRemovesIt(t *testing.T) {
	d := destructiveRig(t, nil)
	out, _ := d.createFolder("Box")
	_, box2 := d.createFolder("Box2")
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box2, "name": "Calc", "type": "kitControl:NumericConst"})
	back := d.rollback(out["batch_id"])
	if back["verdict"] != "verified" || !slices.Equal(d.children(), []string{"Box2"}) {
		t.Errorf("back = %v, children %v", back, d.children())
	}
}

// A non-create rollback keeps its behaviour: later work on the same component does not block it.
func TestRollbackOfASetSlotIgnoresLaterBatchesOnTheComponent(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	set := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	src, _ := d.addOut("Src", "", num(2.0))
	d.Srv.RunWrite("n4_create_link", Args{"source_ord": Folder + "/" + src, "source_slot": "out",
		"target_ord": Folder + "/" + nn, "target_slot": "in10"})
	if back := d.rollback(set["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if v := d.value(h, "out", "value"); v != "1.0" {
		t.Errorf("out = %v", v)
	}
}
