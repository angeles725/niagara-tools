package blackbox

// n4_list_batches and the write branches the audit found untested (port of
// test_tools_destructive.TestListBatches and TestWriteBranchDebt, F13 and F14 of audit
// 2026-10-03).

import (
	"slices"
	"strings"
	"testing"
)

func TestBatchesAreListedNewestFirstWithTheirStateAndRollback(t *testing.T) {
	d := destructiveRig(t, nil)
	a := d.Srv.RunWrite("n4_create_component", createArgs("A"))
	b := d.Srv.RunWrite("n4_create_component", createArgs("B"))
	back := d.rollback(a["batch_id"])
	out := d.Srv.OK("n4_list_batches", nil)
	batches := objects(out["batches"])
	if ids := field(out["batches"], "batch_id"); !same(ids, []any{back["batch_id"], b["batch_id"], a["batch_id"]}) {
		t.Fatalf("ids = %v", ids)
	}
	first := batches[len(batches)-1]
	want := map[string]any{"tool": "n4_create_component", "verdict": "verified", "state": "completed",
		"station_name": "FakeStation", "rolled_back_by": []any{back["batch_id"]}}
	for k, v := range want {
		if !same(first[k], v) {
			t.Errorf("%s = %v, want %v", k, first[k], v)
		}
	}
	if _, ok := first["ts"]; !ok || batches[0]["rollback_of"] != a["batch_id"] {
		t.Errorf("batches = %v", batches)
	}
	if one := objects(d.Srv.OK("n4_list_batches", Args{"limit": 1})["batches"]); len(one) == 0 || one[0]["batch_id"] != back["batch_id"] {
		t.Errorf("limit 1 = %v", one)
	}
	if out["total"] != 3.0 {
		t.Errorf("total = %v", out["total"])
	}
}

// craft journals a completed n4_remove_component batch bid with inverse and targets.
func (d *dRig) craft(bid string, inverse []any, targets map[string]any) {
	d.journalAppend(map[string]any{"batch_id": bid, "phase": "intent", "tool": "n4_remove_component",
		"station_name": "FakeStation", "ops": []any{}, "inverse_plan": inverse, "targets": targets})
	d.journalAppend(map[string]any{"batch_id": bid, "phase": "result", "verdict": "verified",
		"inverse": inverse, "accepted": nil})
}

func folderAdd(name string) map[string]any {
	return map[string]any{"nm": "a", "h": "3", "n": name, "b": map[string]any{"nm": "p", "t": "baja:Folder"}}
}

// Finding 2: op 2 of 2 fails; op 1 was sent and accepted. The error says so, gives the partial
// replies and inverse, and the journal records an in-doubt result.
func TestAMultiOpBatchThatFailsMidWayNamesTheSentOpsAndTheirInverse(t *testing.T) {
	d := destructiveRig(t, nil)
	bid := strings.Repeat("ab", 16)
	d.craft(bid, []any{folderAdd("A"), folderAdd("B")}, map[string]any{Folder: "3"})
	plan := d.Srv.Dry("n4_rollback", Args{"batch_id": bid})
	d.f.set(func(f *faults) {
		f.raw = func(frame map[string]any) (int, []byte) {
			if msgs := objects(frame["m"]); len(msgs) > 0 {
				arg, _ := Dig(msgs[0], "b", "scarg").(map[string]any)
				if ops := objects(arg["ops"]); len(ops) > 0 && ops[0]["nm"] == "a" && ops[0]["n"] == "B" {
					return 500, nil
				}
			}
			return 0, nil
		}
	})
	text := d.Srv.Err("n4_rollback", Execute(Args{"batch_id": bid}, plan["confirmation_token"]))
	d.f.set(func(f *faults) { f.raw = nil })
	if !slices.Equal(d.children(), []string{"A"}) {
		t.Errorf("children = %v", d.children())
	}
	for _, needle := range []string{"in-doubt", "op 2 of 2 failed", "op 1 was sent"} {
		if !strings.Contains(text, needle) {
			t.Errorf("text lacks %q: %s", needle, text)
		}
	}
	if !strings.Contains(strings.ReplaceAll(text, " ", ""), `"nm":"v"`) { // the partial inverse removes A
		t.Errorf("text lacks the inverse: %s", text)
	}
	var batch any
	for _, e := range d.Lines("journal.jsonl") {
		if e["phase"] == "intent" {
			batch = e["batch_id"]
		}
	}
	var results []map[string]any
	for _, e := range d.Lines("journal.jsonl") {
		if e["batch_id"] == batch && e["phase"] == "result" {
			results = append(results, e)
		}
	}
	if len(results) != 1 {
		t.Fatalf("results of %v = %v", batch, results)
	}
	r := results[0]
	if r["in_doubt"] != true || r["sent"] != 1.0 || r["failed_op"] != 1.0 ||
		!same(r["inverse"], []any{map[string]any{"nm": "v", "h": "3", "n": "A"}}) {
		t.Errorf("result = %v", r)
	}
	if state := d.batch(batch)["state"]; state != "in-doubt" {
		t.Errorf("state = %v", state)
	}
	audit := d.Lines("audit.jsonl")
	if last := audit[len(audit)-1]; last["outcome"] != "error" || last["batch_id"] != batch {
		t.Errorf("last audit = %v", last)
	}
}

func TestAMalformedRelinkRecordIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	bid := strings.Repeat("ef", 16)
	d.craft(bid, []any{folderAdd("X"), map[string]any{"relink": map[string]any{"source_path": 1, "source_slot": "out"}}},
		map[string]any{Folder: "3"})
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": bid}); !strings.Contains(text, "malformed relink") {
		t.Errorf("text = %s", text)
	}
	if len(d.children()) != 0 {
		t.Errorf("children = %v", d.children())
	}
}

func TestABatchWithoutRecordedTargetsIsRefused(t *testing.T) {
	d := destructiveRig(t, nil)
	bid := strings.Repeat("fe", 16)
	d.craft(bid, []any{map[string]any{"nm": "v", "h": "3", "n": "x"}}, map[string]any{})
	if text := d.Srv.Err("n4_rollback", Args{"batch_id": bid}); !strings.Contains(text, "does not record which components") {
		t.Errorf("text = %s", text)
	}
}
