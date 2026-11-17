package writeops

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A replication whose add k failed is in-doubt (retro 2026-11-17 mcpn4-replicate-points, Δ5):
// n4_rollback removes only the copies the station accepted (by the name its reply assigned) that
// the read-back finds with the type the batch added; every other copy keeps a per-copy verdict.

const replicaBatch = "0123456789abcdef0123456789abcdef"

// journalReplica journals an n4_replicate_points batch of one "Gw" add per target, as the engine
// does: its intent and, when result is not nil, its result record.
func (r *trig) journalReplica(t *testing.T, targets []*tnode, result *pyjson.Object) {
	t.Helper()
	ops, inverse, handles := []any{}, []any{}, obj()
	for _, tg := range targets {
		ops = append(ops, obj("nm", "a", "h", tg.h, "n", "Gw", "b", obj("nm", "p", "t", "kitControl:NumericConst", "n", "Gw")))
		inverse = append(inverse, obj("nm", "v", "h", tg.h, "n", "Gw"))
		handles.Set(tFolder+"/"+tg.name, tg.h)
	}
	if err := r.state.Journal.Append(obj("batch_id", replicaBatch, "ts", "2026-11-17T00:00:00Z", "tool", "n4_replicate_points",
		"ops", ops, "inverse_plan", inverse, "station_name", "FakeStation", "phase", "intent", "targets", handles)); err != nil {
		t.Fatal(err)
	}
	if result != nil {
		result.Set("batch_id", replicaBatch)
		result.Set("ts", "2026-11-17T00:00:01Z")
		result.Set("phase", "result")
		if err := r.state.Journal.Append(result); err != nil {
			t.Fatal(err)
		}
	}
}

// targets adds a folder per name under Folder.
func (r *trig) targets(names ...string) []*tnode {
	out := []*tnode{}
	for _, n := range names {
		out = append(out, r.st.add(r.st.byH["3"], n, "baja:Folder"))
	}
	return out
}

// partial is the result record of a replication whose add len(replies)+1 failed (inDoubtPartial).
func partial(replies ...any) *pyjson.Object {
	return obj("accepted", replies, "inverse", []any{}, "verdict", "unverified", "in_doubt", true,
		"sent", len(replies), "failed_op", len(replies))
}

func TestAnInDoubtReplicationRollsBackOnlyTheCopiesTheStationAccepted(t *testing.T) {
	r := newTRig(t, 200)
	tg := r.targets("T1", "T2", "T3", "T4")
	r.st.add(tg[0], "Gw1", "kitControl:NumericConst") // op 1: the station renamed it Gw1
	r.st.add(tg[0], "Gw", "kitControl:NumericConst")  // a same-name component op 1 did not make
	r.st.add(tg[1], "Gw", "baja:Folder")              // op 2: accepted, now another type
	r.st.add(tg[2], "Gw", "kitControl:NumericConst")  // op 3 failed: maybe applied
	r.st.add(tg[3], "Gw", "kitControl:NumericConst")  // op 4 was never sent
	r.journalReplica(t, tg, partial([]any{obj("id", "a", "nn", "Gw1")}, []any{obj("id", "a")}))

	text, isErr, dry := r.call(t, "n4_rollback", obj("batch_id", replicaBatch))
	if isErr {
		t.Fatalf("dry run refused: %s", text)
	}
	plan := get(dry, "plan").(*pyjson.Object)
	if ops := pyjson.Repr(get(plan, "ops")); ops != pyjson.Repr([]any{obj("nm", "v", "h", tg[0].h, "n", "Gw1")}) {
		t.Errorf("ops %s: want only the remove of T1/Gw1", ops)
	}
	notes := pyjson.Repr(get(plan, "notes"))
	for _, want := range []string{
		"op 1, " + tFolder + "/T1 'Gw1': found",
		"op 2, " + tFolder + "/T2 'Gw': changed",
		"op 3, " + tFolder + "/T3 'Gw': unverified",
		"op 4, " + tFolder + "/T4 'Gw': not_sent",
	} {
		if !strings.Contains(notes, want) {
			t.Errorf("notes %s lack %q", notes, want)
		}
	}

	text, isErr, out := r.write(t, "n4_rollback", obj("batch_id", replicaBatch))
	if isErr {
		t.Fatal(text)
	}
	if v := get(out, "verdict"); v != "partial" {
		t.Errorf("verdict %v: copies were left in place, want partial", v)
	}
	if tg[0].child("Gw1") != nil || tg[0].child("Gw") == nil || tg[1].child("Gw") == nil ||
		tg[2].child("Gw") == nil || tg[3].child("Gw") == nil {
		t.Errorf("only T1/Gw1 must be removed")
	}
	copies := pyjson.Repr(get(get(out, "observed").(*pyjson.Object), "copies"))
	for _, want := range []string{"'state': 'removed'", "'state': 'changed'", "'state': 'unverified'", "'state': 'not_sent'"} {
		if !strings.Contains(copies, want) {
			t.Errorf("observed copies %s lack %s", copies, want)
		}
	}
	if n := get(get(out, "write_budget").(*pyjson.Object), "executed"); pyjson.Repr(n) != "0" {
		t.Errorf("executed %v: a rollback spends no write", n)
	}
	if text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", replicaBatch)); !isErr || !strings.Contains(text, "already rolled back") {
		t.Errorf("second rollback: %s", text)
	}
}

func TestAnInDoubtReplicationWithNoCopyAcceptedStillRefuses(t *testing.T) {
	r := newTRig(t, 200)
	tg := r.targets("T1", "T2")
	r.st.add(tg[0], "Gw", "kitControl:NumericConst")
	r.journalReplica(t, tg, partial())
	text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", replicaBatch))
	if !isErr || !strings.Contains(text, "none of its copies was accepted by the station") || len(r.st.sent) != 0 {
		t.Errorf("isErr %v: %s", isErr, text)
	}
}

func TestAnInDoubtReplicationWithoutAResultRecordKeepsTheInDoubtRefusal(t *testing.T) {
	r := newTRig(t, 200)
	tg := r.targets("T1")
	r.st.add(tg[0], "Gw", "kitControl:NumericConst")
	r.journalReplica(t, tg, nil)
	text, isErr, _ := r.call(t, "n4_rollback", obj("batch_id", replicaBatch))
	if !isErr || !strings.Contains(text, "is in-doubt (the station may or may not have applied it); rolling it back blindly") {
		t.Errorf("isErr %v: %s", isErr, text)
	}
}

// Δ6: every n4_rollback is outside --max-writes, like n4_save_station: a rollback of a replication
// runs with no write left and leaves the count where it was.
func TestARollbackRunsWithTheBudgetSpentAndSpendsNoWrite(t *testing.T) {
	r := newTRig(t, 2)
	tg := r.targets("T1", "T2", "T3")
	inverse := []any{}
	for _, n := range tg {
		r.st.add(n, "Gw", "kitControl:NumericConst")
		inverse = append(inverse, obj("nm", "v", "h", n.h, "n", "Gw"))
	}
	r.journalReplica(t, tg, obj("accepted", []any{}, "inverse", inverse, "verdict", "verified"))
	r.state.WritesExecuted = 2
	text, isErr, out := r.write(t, "n4_rollback", obj("batch_id", replicaBatch))
	if isErr || get(out, "verdict") != "verified" {
		t.Fatalf("%s", text)
	}
	if got := pyjson.Repr(get(out, "write_budget")); got != "{'executed': 2, 'max': 2, 'remaining': 0, 'save_counts': False, 'rollback_counts': False}" {
		t.Errorf("write_budget %s: the rollback must not spend a write", got)
	}
	for _, n := range tg {
		if n.child("Gw") != nil {
			t.Errorf("%s/Gw was not removed", n.name)
		}
	}
}
