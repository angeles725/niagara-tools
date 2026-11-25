package blackbox

// The batches a chain=true rollback removed with a create stop refusing a later rollback of an
// ancestor create (issue #1058): driven through the real execute path, so a change in what a rollback
// journals (rollback_of, state, verdict) cannot silently disable vanishedWith.

import (
	"slices"
	"testing"
)

func TestAChainRollbackFreesTheAncestorCreateOfTheBatchesItRemoved(t *testing.T) {
	d := destructiveRig(t, nil)
	outer, outerOrd := d.createFolder("Outer")
	box := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": outerOrd, "name": "Box", "type": "baja:Folder"})
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": outerOrd + "/Box", "name": "Calc", "type": "kitControl:NumericConst"})

	dry := d.Srv.Dry("n4_rollback", chainArgs(box["batch_id"].(string)))
	back := d.Srv.OK("n4_rollback", Execute(chainArgs(box["batch_id"].(string)), dry["confirmation_token"]))
	if back["verdict"] != "verified" || d.node("Outer", "Box") != nil {
		t.Fatalf("chain rollback: %v", back)
	}

	id := outer["batch_id"].(string)
	plain := d.Srv.Dry("n4_rollback", Args{"batch_id": id})
	if plain["confirmation_token"] == nil {
		t.Fatalf("the ancestor create still refuses after the chain rollback: %v", plain)
	}
	d.Srv.OK("n4_rollback", Execute(Args{"batch_id": id}, plain["confirmation_token"]))
	if slices.Contains(d.children(), "Outer") {
		t.Errorf("Outer was not removed: %v", d.children())
	}
}
