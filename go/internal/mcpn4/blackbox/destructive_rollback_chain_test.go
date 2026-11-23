package blackbox

// n4_rollback chain=true (issue #930): a create batch with later batches under it is removed in one
// step, under one token, the later batches named newest first. Without chain the refusal is unchanged.

import (
	"slices"
	"testing"
)

// chainRig is a created Box with a later create of Calc under it and a link from outside into
// Calc; it returns the rig, the create's batch id and the later batch ids (oldest first).
func chainRig(t *testing.T) (*dRig, string, []string) {
	t.Helper()
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	block := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	src, _ := d.addOut("Src", "", num(1.0))
	link := d.Srv.RunWrite("n4_create_link", Args{"source_ord": Folder + "/" + src, "source_slot": "out",
		"target_ord": box + "/Calc", "target_slot": "in10"})
	return d, out["batch_id"].(string), []string{block["batch_id"].(string), link["batch_id"].(string)}
}

func chainArgs(id string) Args { return Args{"batch_id": id, "chain": true} }

func TestRollbackChainRemovesACreateWithItsLaterBatchesUnderOneToken(t *testing.T) {
	d, id, later := chainRig(t)
	dry := d.Srv.Dry("n4_rollback", chainArgs(id))
	note := "chain: removes the created " + Folder + "/Box with the later batch(es) under it, newest first: " +
		later[1] + " (n4_create_link), " + later[0] + " (n4_create_component)"
	if len(noteWith(planOf(dry)["notes"], note)) == 0 {
		t.Errorf("notes = %v, want %q", planOf(dry)["notes"], note)
	}
	back := d.Srv.OK("n4_rollback", Execute(chainArgs(id), dry["confirmation_token"]))
	if back["verdict"] != "verified" || slices.Contains(d.children(), "Box") || !slices.Contains(d.children(), "Src") {
		t.Errorf("back = %v, children %v", back, d.children())
	}
}

// The token of a chain dry run is not usable by a call without chain: that call is the plain rollback
// and its refusal stands.
func TestRollbackChainTokenDoesNotCarryOverToACallWithoutChain(t *testing.T) {
	d, id, _ := chainRig(t)
	dry := d.Srv.Dry("n4_rollback", chainArgs(id))
	text := d.Srv.Err("n4_rollback", Execute(Args{"batch_id": id}, dry["confirmation_token"]))
	wantAll(t, text, "later batch(es) wrote under it", "newest first")
	if d.node("Box") == nil {
		t.Errorf("Box was removed")
	}
}

// The refusal without chain names chain=true as the way out, and stays otherwise as it was.
func TestRollbackWithoutChainStillRefusesAndPointsToChain(t *testing.T) {
	d, id, _ := chainRig(t)
	text := d.refusedRollback(id)
	wantAll(t, text, "later batch(es) wrote under it", "newest first", "n4_rollback chain=true")
}

// A later batch that also wrote outside the created component is not removed with it.
func TestRollbackChainRefusesALaterBatchThatWroteOutside(t *testing.T) {
	d := destructiveRig(t, nil)
	out, box := d.createFolder("Box")
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": box, "name": "Calc", "type": "kitControl:NumericConst"})
	src, _ := d.addOut("Src", "", num(1.0))
	dst, _ := d.addOut("Dst", "", num(1.0))
	link := func(target string) Args {
		return Args{"source_ord": Folder + "/" + src, "source_slot": "out", "target_ord": target, "target_slot": "in10"}
	}
	both := d.Srv.RunWrite("n4_create_link", Args{"links": []any{link(box + "/Calc"), link(Folder + "/" + dst)}})
	before := d.rollbackBatches()
	text := d.Srv.Err("n4_rollback", chainArgs(out["batch_id"].(string)))
	want := "batch " + both["batch_id"].(string) + " (n4_create_link) also wrote " + Folder + "/" + dst + ", outside " + box +
		": roll it back first, chain=true only removes batches that stay under the created component: refusing"
	if text != want {
		t.Errorf("error = %q, want %q", text, want)
	}
	if d.rollbackBatches() != before || d.node("Box", "Calc") == nil {
		t.Errorf("a refused chain changed something")
	}
}

// A child no batch made still refuses the chain: the journal cannot vouch for it.
func TestRollbackChainStillRefusesAChildNoBatchMade(t *testing.T) {
	d, id, _ := chainRig(t)
	d.Add(d.node("Box").Handle, "Manual", "")
	text := d.Srv.Err("n4_rollback", chainArgs(id))
	wantAll(t, text, "it holds child(ren) or link(s) its create did not make: Manual")
	if d.node("Box") == nil {
		t.Errorf("Box was removed")
	}
}

// chain on a batch that is not a create has nothing to chain: it rolls back as before, with a note.
func TestRollbackChainOnANonCreateBatchRollsBackAsBefore(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	set := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	id := set["batch_id"].(string)
	dry := d.Srv.Dry("n4_rollback", chainArgs(id))
	if len(noteWith(planOf(dry)["notes"], "chain: batch "+id+" is not a create: nothing to chain")) == 0 {
		t.Errorf("notes = %v", planOf(dry)["notes"])
	}
	if back := d.Srv.OK("n4_rollback", Execute(chainArgs(id), dry["confirmation_token"])); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if v := d.value(h, "out", "value"); v != "1.0" {
		t.Errorf("out = %v", v)
	}
}
