package blackbox

// Probes of the independent verifier of #809: the scope-root rollback exception is for creates only.

import (
	"strings"
	"testing"
)

func rootHandle(d *dRig) string {
	if n := d.node("Root"); n != nil {
		return n.Handle
	}
	return ""
}

func toRootScope(d *dRig) {
	d.Restart(func(o *Options) { o.WriteScopes = []string{scopeRoot} })
	d.ConnectVerified()
}

// Verifier probe P1 of #809: create of Root under the wide scope, hand-made child inside, rollback under the root scope.
func TestACreateRollbackUnderTheRootScopeKeepsAHandMadeChild(t *testing.T) {
	d := destructiveRig(t, nil)
	d.ConnectVerified()
	out := d.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	d.Add(rootHandle(d), "Hand", "baja:Folder")
	toRootScope(d)
	r := d.Srv.Call("n4_rollback", Args{"batch_id": out["batch_id"]})
	t.Logf("create rollback: isErr=%v text=%.300s", r.IsError, r.Text)
	if !r.IsError || !strings.Contains(r.Text, "Hand") || d.node("Root") == nil {
		t.Errorf("the create rollback must be refused by the later-work check naming Hand: %v %.300s", r.IsError, r.Text)
	}
}

// Verifier probe P2 of #809 (blocker): only a create of the scope root is undone under that scope;
// a copy makes Root under the wide scope; a hand-made child is added inside; rollback under the root scope.
func TestACopyRollbackUnderTheRootScopeIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	d.ConnectVerified()
	d.Add("3", "Src", "baja:Folder")
	cp := d.Srv.RunWrite("n4_copy_component", Args{"parent_ord": Folder, "name": "Src", "new_names": []any{"Root"}})
	d.Add(rootHandle(d), "Hand", "baja:Folder")
	toRootScope(d)
	dry := d.Srv.Call("n4_rollback", Args{"batch_id": cp["batch_id"]})
	t.Logf("copy rollback dry: isErr=%v text=%.400s", dry.IsError, dry.Text)
	if !dry.IsError || !strings.Contains(dry.Text, "outside the write scope") {
		t.Errorf("copy rollback under the root scope: isErr=%v text=%.300s", dry.IsError, dry.Text)
	}
	if dry.IsError {
		return
	}
	back := d.Srv.Call("n4_rollback", Execute(Args{"batch_id": cp["batch_id"]}, d.Srv.Dry("n4_rollback", Args{"batch_id": cp["batch_id"]})["confirmation_token"]))
	t.Logf("copy rollback exec: isErr=%v text=%.300s root=%v", back.IsError, back.Text, d.node("Root") != nil)
	if d.node("Root") == nil {
		t.Errorf("BLOCKER: the root scope deleted the populated root (hand-made child Hand) via rollback of a copy batch")
	}
}

// Verifier probe P2b of #809 (blocker): copy of Root plus a later journaled create inside (under the root scope), then rollback of the copy.
func TestACopyRollbackUnderTheRootScopeIsRefusedAfterLaterWork(t *testing.T) {
	d := destructiveRig(t, nil)
	d.ConnectVerified()
	d.Add("3", "Src", "baja:Folder")
	cp := d.Srv.RunWrite("n4_copy_component", Args{"parent_ord": Folder, "name": "Src", "new_names": []any{"Root"}})
	toRootScope(d)
	d.Srv.RunWrite("n4_create_component", Args{"parent_ord": scopeRoot, "name": "Inner", "type": "baja:Folder"})
	r := d.Srv.Call("n4_rollback", Args{"batch_id": cp["batch_id"]})
	t.Logf("copy rollback with later journaled work: isErr=%v text=%.300s", r.IsError, r.Text)
	if !r.IsError || !strings.Contains(r.Text, "outside the write scope") {
		t.Errorf("copy rollback with later work inside: %v %.300s", r.IsError, r.Text)
	}
}
