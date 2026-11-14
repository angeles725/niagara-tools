package blackbox

// n4_rollback of each write tool's batch (port of test_tools_destructive.TestRollback, the
// per-tool inverses and the compare-before-restore guard).

import (
	"slices"
	"strings"
	"testing"
)

func createArgs(name string) Args {
	return Args{"parent_ord": Folder, "name": name, "type": "kitControl:NumericConst"}
}

// setOut is n4_set_slot of comp's out to 9.0 (a StatusNumeric).
func setOut(comp string) Args {
	return Args{"ord": Folder + "/" + comp, "slot": "out", "value": 9.0, "value_type": "baja:StatusNumeric"}
}

// invokeSet is n4_invoke_action set 7.0 on comp.
func invokeSet(comp string) Args {
	return Args{"ord": Folder + "/" + comp, "action": "set", "arg": 7.0, "arg_type": "baja:Double"}
}

func TestRollbackOfACreateRemovesTheComponentAsALinkedBatch(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	reply := d.Srv.Dry("n4_rollback", Args{"batch_id": out["batch_id"]})
	if ops := planOf(reply)["ops"]; !same(ops, []any{map[string]any{"nm": "v", "h": "3", "n": "Pump"}}) {
		t.Errorf("ops = %v", ops)
	}
	if !slices.Equal(d.children(), []string{"Pump"}) {
		t.Errorf("children after the dry run = %v", d.children())
	}
	back := d.Srv.OK("n4_rollback", Execute(Args{"batch_id": out["batch_id"]}, reply["confirmation_token"]))
	if back["verdict"] != "verified" || len(d.children()) != 0 {
		t.Errorf("back = %v, children %v", back, d.children())
	}
	if view := d.batch(back["batch_id"]); view["rollback_of"] != out["batch_id"] || view["tool"] != "n4_rollback" {
		t.Errorf("journal view = %v", view)
	}
}

func TestRollbackFollowsTheServerAssignedName(t *testing.T) {
	d := destructiveRig(t, nil)
	d.Add("3", "Pump", "")
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	if !slices.Equal(d.children(), []string{"Pump", "Pump1"}) { // the precondition: a renamed create
		t.Fatalf("children = %v", d.children())
	}
	d.rollback(out["batch_id"])
	if !slices.Equal(d.children(), []string{"Pump"}) {
		t.Errorf("children = %v", d.children())
	}
}

func TestRollbackOfASetSlotRestoresThePreviousValue(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	out := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if v := d.value(h, "out", "value"); v != "1.0" {
		t.Errorf("out = %v", v)
	}
}

// F6 (audit 2026-10-03): compare before restore.
func TestASlotChangedAfterTheBatchIsNotOverwrittenByItsRollback(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	out := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	d.Set(h, "out", StatusNumeric(4.0, "0")) // an operator changes it
	text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	for _, needle := range []string{"changed since batch", "4.0", "9.0"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if v := d.value(h, "out", "value"); v != "4.0" || len(d.rollbacksOf(out["batch_id"])) != 0 {
		t.Errorf("out = %v, rollbacks %v", v, d.rollbacksOf(out["batch_id"]))
	}
}

func TestASlotChangedBetweenDryRunAndConfirmIsRefusedAtConfirm(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	out := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	plan := d.Srv.Dry("n4_rollback", Args{"batch_id": out["batch_id"]})
	d.Set(h, "out", StatusNumeric(4.0, "0"))
	text := d.Srv.Err("n4_rollback", Execute(Args{"batch_id": out["batch_id"]}, plan["confirmation_token"]))
	if !strings.Contains(text, "changed since batch") || d.value(h, "out", "value") != "4.0" {
		t.Errorf("text %s, out %v", text, d.value(h, "out", "value"))
	}
}

func TestRollbackOfANestedStatusSlotRestoresAndVerifiesIt(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Calc", "", num(1.0))
	sp := StatusNumeric(7.5, "0")
	sp["n"] = "sp"
	d.Set(h, "grp", map[string]any{"nm": "p", "t": "baja:Struct", "s": []any{sp}})
	args := setOut(nn)
	args["slot"] = "grp/sp"
	out := d.Srv.RunWrite("n4_set_slot", args)
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if v := d.value(h, "grp", "sp", "value"); v != "7.5" {
		t.Errorf("grp/sp = %v", v)
	}
}

func TestAFallbackChangedAfterInvokeSetIsNotOverwritten(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Sp", "control:NumericWritable", nil)
	out := d.Srv.RunWrite("n4_invoke_action", invokeSet(nn))
	d.Set(h, "fallback", StatusNumeric(3.0, "0"))
	text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	if !strings.Contains(text, "changed since batch") || d.value(h, "fallback", "value") != "3.0" {
		t.Errorf("text %s, fallback %v", text, d.value(h, "fallback", "value"))
	}
}

func TestRollbackOfALinkRemovesIt(t *testing.T) {
	d := destructiveRig(t, nil)
	src, _ := d.addOut("Src", "", num(1.0))
	tgt, tgtH := d.addOut("Tgt", "", nil)
	out := d.Srv.RunWrite("n4_create_link", Args{"source_ord": Folder + "/" + src, "source_slot": "out",
		"target_ord": Folder + "/" + tgt, "target_slot": "in10"})
	if d.node(tgt, "Link") == nil { // the precondition: the link exists
		t.Fatalf("no link on %s: %v", tgt, d.childNames(d.node(tgt)))
	}
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if d.node(tgt, "Link") != nil {
		t.Errorf("the link of %s survived: %v", tgtH, d.childNames(d.node(tgt)))
	}
}

func TestRollbackOfInvokeSetRestoresTheFallback(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, h := d.addOut("Sp", "control:NumericWritable", nil)
	out := d.Srv.RunWrite("n4_invoke_action", invokeSet(nn))
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if v := d.value(h, "fallback", "value"); v != "0.0" {
		t.Errorf("fallback = %v", v)
	}
}

// Live finding 2 (2026-10-01): `station:|slot:/` + `/` made `station:|slot://X`.
func TestAFolderRemovedAtTheStationRootRollsBack(t *testing.T) {
	d := destructiveRig(t, nil)
	d.restart(func(o *Options) { o.WriteScopes = []string{destructiveStationRootOr} })
	top := d.Add("2", "RootGrp", "baja:Folder")
	d.Add(top.Handle, "Src", "")
	removed := d.Srv.RunWrite("n4_remove_component", Args{"parent_ord": destructiveStationRootOr, "name": top.Name})
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	restored := d.nodeFrom("2", top.Name, "Src") != nil
	if !restored {
		t.Errorf("%s/Src was not re-created", top.Name)
	}
}
