package blackbox

// The rollback of a remove: what the re-created subtree holds and the order it is added in
// (port of test_tools_destructive.TestRollback, the remove inverse).

import (
	"strings"
	"testing"
)

// srcHandle is the handle of the group's Src.
func (d *dRig) srcHandle(nn string) string { return d.node(nn, "Src").Handle }

// addOf is the first add op of ops named name, nil when absent.
func addOf(ops []map[string]any, name string) map[string]any {
	for _, op := range adds(ops) {
		if op["n"] == name {
			return op
		}
	}
	return nil
}

func TestRollbackOfARemoveRecreatesTypeSlotsAnnotationAndLinks(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	grp := d.node(nn)
	if grp == nil {
		t.Fatalf("%s was not re-created", nn)
	}
	if grp.Type != "baja:Folder" || d.value(grp.Handle, "wsAnnotation") != "1,2,3" {
		t.Errorf("group type %s, annotation %v", grp.Type, d.value(grp.Handle, "wsAnnotation"))
	}
	if d.node(nn, "Src", "out") != nil { // runtime output: not restored
		t.Errorf("Src.out came back")
	}
	if v := d.value(grp.Handle, "Tgt", "Link", "targetSlotName"); v != "in10" {
		t.Errorf("Tgt link target slot = %v", v)
	}
	if !same(back["relinks"], map[string]any{"restored": 1, "skipped": 0}) {
		t.Errorf("relinks = %v", back["relinks"])
	}
	tgtH := d.node(nn, "Tgt").Handle
	want := []any{map[string]any{"nm": "v", "h": tgtH, "n": "Link"}, map[string]any{"nm": "v", "h": "3", "n": nn}}
	if inv := d.batch(back["batch_id"])["inverse"]; !same(inv, want) {
		t.Errorf("inverse = %v, want %v", inv, want)
	}
}

func TestTheSnapshotKeepsConfigurationAndDropsRuntimeOutputSlots(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	src := d.srcHandle(nn)
	d.Set(src, "fallback", StatusNumeric(7.0, "0;activeLevel=e_def"))
	d.Set(src, "out", StatusNumeric(4.5, "0;activeLevel=e_in16"))
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	d.rollback(removed["batch_id"])
	op := addOf(sent(), "Src")
	slots := field(Dig(op, "b", "s"), "n")
	if op == nil || destructiveContainsAny(slots, "out") || !destructiveContainsAny(slots, "fallback") {
		t.Errorf("Src add = %v", op)
	}
	fallback := d.node(nn, "Src", "fallback")
	if fallback == nil {
		t.Fatal("Src.fallback was not re-created")
	}
	if got := d.childValues(nn, "Src", "fallback"); got["value"] != "7.0" || got["status"] != "0" { // no runtime activeLevel facet
		t.Errorf("fallback = %v", got)
	}
}

// Review R4-status-coercion: a null (0x40) fallback must not come back as ok.
func TestTheSnapshotKeepsConfiguredStatusBitsAndDropsOnlyFacets(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	d.Set(d.srcHandle(nn), "fallback", StatusNumeric(7.0, "40;activeLevel=e_def"))
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	d.rollback(removed["batch_id"])
	if got := d.childValues(nn, "Src", "fallback"); got["status"] != "40" { // null bit kept, facet dropped
		t.Errorf("fallback = %v", got)
	}
}

func TestGrandchildrenAreAddedAfterTheirParentBreadthFirst(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, gh := d.group("Grp")
	sub := d.Add(gh, "Sub", "baja:Folder")
	d.Add(sub.Handle, "Deep", "")
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if got := names(adds(sent())); !same(got, []any{nn, "Src", "Tgt", "Sub", "Deep"}) {
		t.Errorf("adds = %v", got)
	}
	if d.node(nn, "Sub", "Deep") == nil {
		t.Errorf("Sub/Deep was not re-created")
	}
}

func TestThePlanListsComponentsByPathAndItsHashIsStable(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	first := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	second := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	if first["plan_hash"] == nil || first["plan_hash"] != second["plan_hash"] {
		t.Errorf("plan hashes %v and %v", first["plan_hash"], second["plan_hash"])
	}
	comps := objects(planOf(first)["components"])
	pairs := []any{}
	for _, c := range comps {
		pairs = append(pairs, []any{c["parent_path"], c["n"]})
	}
	if !same(pairs, []any{[]any{"", "Src"}, []any{"", "Tgt"}}) {
		t.Fatalf("components = %v", comps)
	}
	if _, ok := comps[0]["h"]; ok {
		t.Errorf("a component carries a handle: %v", comps[0])
	}
	if ops := objects(planOf(first)["ops"]); len(ops) == 0 || ops[0]["h"] != "3" {
		t.Errorf("ops = %v", ops)
	}
}

func TestRollbackAddsOneComponentPerOpParentsFirstWithTheNewHandle(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	sent := d.f.record()
	d.rollback(removed["batch_id"])
	ops := adds(sent())
	if !same(names(ops), []any{nn, "Src", "Tgt"}) {
		t.Fatalf("adds = %v", names(ops))
	}
	for _, op := range ops { // only plain slots and wsAnnotation
		for _, c := range objects(Dig(op, "b", "s")) {
			if typ, _ := c["t"].(string); !strings.HasPrefix(typ, "baja:") {
				t.Errorf("add %v nests %v", op["n"], c)
			}
		}
	}
	grp := d.node(nn)
	if ops[0]["h"] != "3" || ops[1]["h"] != grp.Handle || ops[2]["h"] != grp.Handle {
		t.Errorf("add handles %v %v %v, group %s", ops[0]["h"], ops[1]["h"], ops[2]["h"], grp.Handle)
	}
	if first := objects(Dig(ops[0], "b", "s")); len(first) == 0 || first[0]["n"] != "wsAnnotation" {
		t.Errorf("group add = %v", ops[0])
	}
}

func destructiveContainsAny(list []any, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
