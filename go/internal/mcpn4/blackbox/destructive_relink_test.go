package blackbox

// Relinks are first-class planned ops (port of test_tools_destructive.TestRollback,
// R3-relink-unjournaled-writes).

import (
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestRelinksAreJournaledWithIntentBeforeTheSendAndAResult(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	back := d.rollback(removed["batch_id"])
	var phases []any
	var relinkIntent map[string]any
	for _, e := range d.Lines("journal.jsonl") {
		if e["batch_id"] == back["batch_id"] {
			phases = append(phases, e["phase"])
		}
		if e["phase"] == "relink-intent" && relinkIntent == nil {
			relinkIntent = e
		}
	}
	if !same(phases, []any{"intent", "component-intent", "component-intent", "relink-intent", "result"}) {
		t.Errorf("phases = %v", phases)
	}
	ops := objects(relinkIntent["ops"])
	if len(ops) != 1 || ops[0]["ssc"] != "checkLinks" || Dig(ops[0], "arg", "ss") != "out" {
		t.Fatalf("relink intent = %v", relinkIntent)
	}
	view := d.batch(back["batch_id"])
	if !same(view["relink_ops"], relinkIntent["ops"]) || !same(view["relinks"], map[string]any{"restored": 1, "skipped": 0}) {
		t.Errorf("journal view = %v", view)
	}
}

func TestTheRelinkIntentIsOnDiskBeforeTheRelinkIsSent(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	var mu sync.Mutex
	var seen [][]any
	d.f.set(func(f *faults) {
		f.before = func(sck string) {
			if sck != "checkLinks" {
				return
			}
			phases := []any{}
			for _, e := range d.journalEntries() {
				phases = append(phases, e["phase"])
			}
			mu.Lock()
			seen = append(seen, phases)
			mu.Unlock()
		}
	})
	d.rollback(removed["batch_id"])
	mu.Lock()
	defer mu.Unlock()
	if len(seen) == 0 || !slices.Contains(seen[len(seen)-1], any("relink-intent")) {
		t.Errorf("journal phases at each relink: %v", seen)
	}
}

func TestARollbackWhoseRelinksExceedTheBudgetIsRefusedBeforeSending(t *testing.T) {
	d := destructiveRig(t, nil)
	d.restart(func(o *Options) { o.MaxWrites = 2 })
	nn, removed := d.removeGroup()
	text := d.Srv.Err("n4_rollback", Args{"batch_id": removed["batch_id"]})
	if !strings.Contains(text, "--max-writes") || !strings.Contains(text, "relink") {
		t.Errorf("text = %s", text)
	}
	if slices.Contains(d.children(), nn) { // nothing was re-created
		t.Errorf("children = %v", d.children())
	}
}

func TestRelinkSpecsAreHashedByPathNotByHandle(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	plan := planOf(d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]}))
	if !same(plan["relinks"], []any{map[string]any{"source_path": "Src", "source_slot": "out",
		"target_path": "Tgt", "target_slot": "in10"}}) {
		t.Errorf("relinks = %v", plan["relinks"])
	}
}

func TestTheRollbackJournalsAnInverseThatRemovesTheLinksItCreated(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	back := d.rollback(removed["batch_id"])
	inverse := objects(d.batch(back["batch_id"])["inverse"])
	if len(inverse) == 0 || !slices.Contains(d.children(), nn) {
		t.Fatalf("inverse %v, children %v", inverse, d.children())
	}
	for _, op := range inverse { // what undoing the rollback means, link first
		if _, err := d.sync(op); err != nil {
			t.Fatalf("%v: %v", op, err)
		}
	}
	if slices.Contains(d.children(), nn) {
		t.Errorf("children = %v", d.children())
	}
}

func TestAnAmbiguousRelinkReplyMarksTheBatchInDoubt(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	d.f.set(func(f *faults) {
		f.replace = func(sck string, _ map[string]any) (any, error, bool) { // no verdict, no link name
			return []any{[]any{map[string]any{"r": "huh"}}}, nil, sck == "checkLinks"
		}
	})
	back := d.rollback(removed["batch_id"])
	if !same(back["relinks"], map[string]any{"restored": 0, "skipped": 1, "ambiguous": 1}) {
		t.Errorf("relinks = %v", back["relinks"])
	}
	if state := d.batch(back["batch_id"])["state"]; state != "in-doubt" {
		t.Errorf("state = %v", state)
	}
}
