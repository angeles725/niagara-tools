package blackbox

// n4_remove_component (port of test_tools_destructive.TestRemoveComponent).

import (
	"slices"
	"strings"
	"testing"
)

func TestPlanSnapshotsTheSubtreeAsDataAndSendsNothing(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))
	if !same(plan["ops"], []any{map[string]any{"nm": "v", "h": "3", "n": nn}}) {
		t.Errorf("ops = %v", plan["ops"])
	}
	if !slices.Contains(d.children(), nn) {
		t.Errorf("the dry run removed %s: %v", nn, d.children())
	}
	inverse := objects(plan["inverse"])
	if len(inverse) != 2 {
		t.Fatalf("inverse = %v", plan["inverse"])
	}
	add, relink := inverse[0], inverse[1]
	if b, _ := add["b"].(map[string]any); add["nm"] != "a" || add["h"] != "3" || add["n"] != nn || b["t"] != "baja:Folder" {
		t.Errorf("add = %v", add)
	}
	slots := byName(Dig(add, "b", "s"))
	if slots["wsAnnotation"]["v"] != "1,2,3" {
		t.Errorf("wsAnnotation = %v", slots["wsAnnotation"])
	}
	if got := field(slots["Src"]["s"], "n"); len(got) != 0 { // runtime out: dropped
		t.Errorf("Src slots = %v", got)
	}
	if _, ok := byName(slots["Tgt"]["s"])["Link"]; ok { // links are separate
		t.Errorf("Tgt carries its Link: %v", slots["Tgt"])
	}
	if _, ok := slots["Src"]["h"]; ok { // handles are not data
		t.Errorf("Src carries a handle: %v", slots["Src"])
	}
	if !same(relink["relink"], map[string]any{"source_path": "Src", "source_slot": "out",
		"target_path": "Tgt", "target_slot": "in10"}) {
		t.Errorf("relink = %v", relink)
	}
	if len(noteWith(plan["notes"], "depth 3", "not a full restore")) == 0 {
		t.Errorf("notes = %v", plan["notes"])
	}
}

func TestLinksFromOutsideTheSubtreeAreReportedAsNotRestorable(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	other := d.AddWithOut("Other", 1.0)
	d.Link(other, "out", Component{"Tgt", d.node(nn, "Tgt").Handle}, "in11")
	plan := planOf(d.Srv.Dry("n4_remove_component", removeArgs(nn)))
	relinks := 0
	for _, e := range objects(plan["inverse"]) {
		if _, ok := e["relink"]; ok {
			relinks++
		}
	}
	if relinks != 1 || len(noteWith(plan["notes"], "outside", "not restored")) == 0 {
		t.Errorf("relinks %d, notes %v", relinks, plan["notes"])
	}
}

func TestExecuteRemovesVerifiesAndJournalsTheSnapshotAsTheInverse(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	out := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	if out["verdict"] != "verified" || slices.Contains(d.children(), nn) {
		t.Errorf("out = %v, children %v", out, d.children())
	}
	view := d.batch(out["batch_id"])
	if inv := objects(view["inverse"]); len(inv) == 0 || inv[0]["nm"] != "a" || view["state"] != "completed" {
		t.Errorf("journal view = %v", view)
	}
}

func TestMissingChildBadNameAndScopeAreRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	if text := d.Srv.Err("n4_remove_component", removeArgs("Nope")); !strings.Contains(text, "not found") {
		t.Errorf("missing child: %s", text)
	}
	d.Add("3", "Keep", "") // a non-empty precondition: something that could be removed
	before := d.children()
	text := d.Srv.Err("n4_remove_component", removeArgs("../x"))
	if !strings.Contains(text, "name must be a plain slot name") || !strings.Contains(text, "'../x'") {
		t.Errorf("bad name: %s", text)
	}
	if !slices.Equal(d.children(), before) || len(d.Lines("journal.jsonl")) != 0 { // nothing removed or journaled
		t.Errorf("children %v (before %v), journal %v", d.children(), before, d.Lines("journal.jsonl"))
	}
	text = d.Srv.Err("n4_remove_component", Args{"parent_ord": destructiveStationRootOr, "name": "Folder"})
	if !strings.Contains(text, "outside the write scope") {
		t.Errorf("scope: %s", text)
	}
}

func TestReadBackReportsAChildThatIsStillThere(t *testing.T) {
	d := destructiveRig(t, nil)
	d.Add("3", "Keep", "")
	plan := d.Srv.Dry("n4_remove_component", removeArgs("Keep"))
	d.f.set(func(f *faults) {
		f.replace = func(sck string, arg map[string]any) (any, error, bool) {
			ops := objects(arg["ops"])
			return []any{}, nil, sck == "syncTo" && len(ops) == 1 && ops[0]["nm"] == "v"
		}
	})
	out := d.Srv.OK("n4_remove_component", Execute(removeArgs("Keep"), plan["confirmation_token"]))
	if out["verdict"] != "mismatch" {
		t.Errorf("verdict = %v", out["verdict"])
	}
}
