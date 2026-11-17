package blackbox

// Edge probes of the scope-root and link-target rules, from the independent verifier of T4.

import (
	"strings"
	"testing"
)

func TestScopeRootRuleAdmitsNoNeighbourName(t *testing.T) {
	r := rootScopeRig(t)
	for _, n := range []string{"Root/", "Root/X", "..", "", "Root|slot:x", "Root "} {
		if text := r.Srv.Err("n4_create_component", Args{"parent_ord": Folder, "name": n, "type": "baja:Folder"}); !strings.Contains(text, "outside the write scope") {
			t.Errorf("%q: %s", n, text)
		}
	}
	for _, p := range []string{Folder + "/.", Folder + "|slot:", "station:|slot:/Other/../Folder"} {
		r.Srv.Err("n4_create_component", Args{"parent_ord": p, "name": "Root", "type": "baja:Folder"})
	}
	if len(r.Children()) != 0 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("children %v journal %d", r.Children(), len(r.Lines("journal.jsonl")))
	}
}

func TestScopeRootUnderTheStationRootIsNamedWithOneSlash(t *testing.T) {
	r := verifiedRig(t) // scope Folder, which exists
	text := r.Srv.Err("n4_create_component", Args{"parent_ord": "station:|slot:/", "name": "Folder", "type": "baja:Folder"})
	if !strings.Contains(text, "'station:|slot:/Folder' already exists") {
		t.Errorf("refusal = %s", text)
	}
}

func TestScopeRootThatAppearsAfterTheDryRunIsRefusedAtExecute(t *testing.T) {
	r := rootScopeRig(t)
	args := Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"}
	tok := r.Srv.Dry("n4_create_component", args)["confirmation_token"]
	r.Add("3", "Root", "baja:Folder")
	if text := r.Srv.Err("n4_create_component", Execute(args, tok)); !strings.Contains(text, "already exists") {
		t.Errorf("refusal = %s", text)
	}
	if len(r.Children()) != 1 || len(r.Lines("journal.jsonl")) != 0 {
		t.Errorf("children %v journal %d", r.Children(), len(r.Lines("journal.jsonl")))
	}
}

// The same scope rolls its root create back (the rollback undoes only what that batch created, checked on
// the created path as the create was), but not once later work was written inside the root, and
// n4_remove_component of the root stays refused: a scope never deletes itself and its contents in one call.
func TestTheSameScopeRollsBackItsRootCreateOnlyWhileNothingWasWrittenInside(t *testing.T) {
	r := rootScopeRig(t)
	out := r.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	r.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder + "/Root", "name": "Inner", "type": "baja:Folder"})
	if text := r.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]}); !strings.Contains(text, "later") {
		t.Errorf("rollback with later work inside: %s", text)
	}
	if text := r.Srv.Err("n4_remove_component", Args{"parent_ord": Folder, "name": "Root"}); !strings.Contains(text, "outside the write scope") {
		t.Errorf("remove of the root: %s", text)
	}
	fresh := rootScopeRig(t)
	first := fresh.Srv.RunWrite("n4_create_component", Args{"parent_ord": Folder, "name": "Root", "type": "baja:Folder"})
	back := fresh.Srv.RunWrite("n4_rollback", Args{"batch_id": first["batch_id"]})
	if back["verdict"] != "verified" || len(fresh.Children()) != 0 {
		t.Errorf("rollback of the root create: %v, children %v", back["verdict"], fresh.Children())
	}
}

func TestLinkWithNoScopeAndAChainedTargetAreRefusedOnTheTarget(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = nil })
	r.ConnectVerified()
	r.AddWithOut("A", 1)
	text := r.Srv.Err("n4_create_link", Args{"source_ord": Folder + "/A", "source_slot": "out", "target_ord": Folder + "/A", "target_slot": "in10"})
	if !strings.Contains(text, "no --write-scope") {
		t.Errorf("no scope: %s", text)
	}
	v := verifiedRig(t)
	v.AddWithOut("A", 1)
	text = v.Srv.Err("n4_create_link", Args{"source_ord": Folder + "/A", "source_slot": "out", "target_ord": Folder + "/A|slot:..", "target_slot": "in10"})
	if !strings.HasPrefix(text, "target_ord:") || !strings.Contains(text, "is not a plain station ORD") {
		t.Errorf("chained target: %s", text)
	}
}
