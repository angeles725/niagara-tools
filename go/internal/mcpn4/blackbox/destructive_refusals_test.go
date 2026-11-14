package blackbox

// Which batches n4_rollback refuses, and the failed rollbacks it may retry (port of
// test_tools_destructive.TestRollback, the journal checks).

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

// failedRollbackOf journals a failed rollback of id; accepted adds "accepted": null to its result.
func (d *dRig) failedRollbackOf(id any, accepted bool) {
	bid := strings.Repeat("cd", 16)
	d.journalAppend(map[string]any{"batch_id": bid, "phase": "intent", "rollback_of": id})
	result := map[string]any{"batch_id": bid, "phase": "result", "verdict": "failed", "inverse": []any{}}
	if accepted {
		result["accepted"] = nil
	}
	d.journalAppend(result)
}

func TestAFailedRollbackWhoseOpsWereNotAcceptedMayStillBeRetried(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	d.failedRollbackOf(out["batch_id"], true)
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
}

func TestAFailedRollbackMayBeRetried(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	d.failedRollbackOf(out["batch_id"], false)
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
}

// Issue #448.
func TestAJournalLineWhoseBatchIsUnknownRefusesEveryRollback(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	d.appendJournal([]byte(fmt.Sprintf("{\"batch_id\": \"\xff\"}\n{\"batch_id\": [1], \"phase\": \"intent\", \"rollback_of\": \"%s\"}\n", out["batch_id"])))
	text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	if !strings.Contains(text, reasonJournalUnreadable) || !strings.Contains(text, "line(s) 3 (UnicodeDecodeError), 4 (TypeError)") {
		t.Errorf("text = %s", text)
	}
	if !slices.Equal(d.children(), []string{"Pump"}) {
		t.Errorf("children = %v", d.children())
	}
}

func TestAnUnreadableLineOfAnotherBatchDoesNotBlockARollback(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	other := strings.Repeat("ee", 16)
	d.appendJournal([]byte(fmt.Sprintf("{\"batch_id\": [1], \"phase\": \"intent\"}\n"+
		"{\"batch_id\": \"%s\", \"phase\": \"component-intent\", \"ops\": 7}\n", other)))
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": other}); !strings.Contains(text, reasonJournalUnreadable) {
		t.Errorf("text = %s", text)
	}
}

func TestABatchIdWithATrailingNewlineIsMalformed(t *testing.T) {
	d := destructiveRig(t, nil)
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": strings.Repeat("f", 32) + "\n"}); !strings.Contains(text, "must be a 32-character") {
		t.Errorf("text = %s", text)
	}
}

func TestUnknownBatchIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": strings.Repeat("f", 32)}); !strings.Contains(text, "unknown batch") {
		t.Errorf("text = %s", text)
	}
}

func TestABatchCannotBeRolledBackTwiceNorARollbackRolledBack(t *testing.T) {
	d := destructiveRig(t, nil)
	out := d.Srv.RunWrite("n4_create_component", createArgs("Pump"))
	back := d.rollback(out["batch_id"])
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]}); !strings.Contains(text, "already rolled back") {
		t.Errorf("twice: %s", text)
	}
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": back["batch_id"]}); !strings.Contains(text, "is itself a rollback") {
		t.Errorf("rollback of a rollback: %s", text)
	}
}

func TestInDoubtBatchIsRefusedAndShowsTheIntent(t *testing.T) {
	d := destructiveRig(t, nil)
	plan := d.Srv.Dry("n4_create_component", createArgs("Pump"))
	d.f.set(func(f *faults) {
		f.raw = func(frame map[string]any) (int, []byte) {
			for _, m := range objects(frame["m"]) {
				if Dig(m, "b", "sck") == "syncTo" {
					return 500, []byte("boom")
				}
			}
			return 0, nil
		}
	})
	d.Srv.Err("n4_create_component", Execute(createArgs("Pump"), plan["confirmation_token"]))
	d.f.set(func(f *faults) { f.raw = nil })
	lines := d.Lines("journal.jsonl")
	if len(lines) == 0 {
		t.Fatal("nothing was journaled")
	}
	text := d.Srv.Err("n4_rollback", Args{"batch_id": lines[0]["batch_id"]})
	if !strings.Contains(text, "in-doubt") || !strings.Contains(text, `"n4_create_component"`) { // the intent is shown to the human
		t.Errorf("text = %s", text)
	}
}

func TestNothingToRollBackIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.addOut("Sp", "control:NumericWritable", nil)
	out := d.Srv.RunWrite("n4_invoke_action", Args{"ord": Folder + "/" + nn, "action": "active"})
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]}); !strings.Contains(text, "no inverse") {
		t.Errorf("text = %s", text)
	}
}

func TestABatchFromAnotherStationIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	bid := strings.Repeat("ab", 16)
	d.journalAppend(map[string]any{"batch_id": bid, "phase": "intent", "tool": "n4_set_slot",
		"station_name": "OtherStation", "ops": []any{}, "inverse_plan": []any{}, "targets": map[string]any{Folder: "3"}})
	d.journalAppend(map[string]any{"batch_id": bid, "phase": "result", "verdict": "verified",
		"inverse": []any{map[string]any{"nm": "v", "h": "3", "n": "x"}}, "accepted": nil})
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": bid}); !strings.Contains(text, "OtherStation") {
		t.Errorf("text = %s", text)
	}
}

func TestATargetWhoseHandleChangedIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.addOut("Calc", "", num(1.0))
	out := d.Srv.RunWrite("n4_set_slot", setOut(nn))
	if _, err := d.sync(map[string]any{"nm": "v", "h": "3", "n": nn}); err != nil {
		t.Fatal(err)
	}
	d.addOut(nn, "", num(2.0)) // same ORD, new component
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]}); !strings.Contains(text, "handle") {
		t.Errorf("text = %s", text)
	}
}
