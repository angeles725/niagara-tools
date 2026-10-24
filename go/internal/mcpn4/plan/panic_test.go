package plan_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// A Go panic in a tool is Python's unexpected exception: before the intent it is an audited
// internal error, after it the batch is in-doubt (_process wraps it with _in_doubt).

func panicked(r *rig, impl plan.Impl) {
	r.srv.Handlers["n4_set_slot"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_set_slot", impl)
}

func TestAPanickingPlanIsAnAuditedInternalError(t *testing.T) {
	r := newRig(t)
	impl := writeops.SetSlot
	impl.Plan = func(plan.Client, *pyjson.Object) (*plan.Planned, error) { panic("boom") }
	panicked(r, impl)
	text, isErr, _ := r.call(t, "n4_set_slot", setArgs())
	audit := r.lines(t, "audit.jsonl")
	if !isErr || text != "internal error (Exception)" || len(audit) != 1 ||
		get(audit[0], "outcome") != "error" || get(audit[0], "reason") != "Exception: boom" {
		t.Fatalf("%s %v %s", text, isErr, pyjson.Repr(audit))
	}
}

func TestAPanickingSendLeavesAnInDoubtBatch(t *testing.T) {
	r := newRig(t)
	tok := r.dry(t, "n4_set_slot", setArgs())
	r.st.onSend = func() { panic("boom") }
	text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), tok)
	journal, audit := r.lines(t, "journal.jsonl"), r.lines(t, "audit.jsonl")
	batch := get(journal[0], "batch_id").(string)
	want := "station call failed after the intent was journaled: batch " + batch + " is in-doubt (the station " +
		"may or may not have applied it; inspect it before retrying): Exception"
	if !isErr || text != want || len(journal) != 1 || len(r.st.sent) != 1 {
		t.Fatalf("%s %v %s", text, isErr, pyjson.Repr(journal))
	}
	last := audit[len(audit)-1]
	if get(last, "outcome") != "error" || get(last, "batch_id") != batch || get(last, "reason") != want {
		t.Fatal(pyjson.Repr(last))
	}
}

// The read-back, inverse and recover steps catch every exception (_execute): a panic there is
// a result, not an in-doubt error.
func TestAPanickingReadBackIsAFailedVerdict(t *testing.T) {
	r := newRig(t)
	impl := writeops.SetSlot
	impl.Readback = func(plan.Client, *pyjson.Object, *plan.Planned, []any, []any) (any, any, any, string, error) {
		panic("boom")
	}
	panicked(r, impl)
	_, isErr, out := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	result := r.lines(t, "journal.jsonl")[1]
	if isErr || get(out, "verdict") != "failed" || get(out, "readback_error") != "Exception: boom" ||
		get(result, "readback_failed") != true {
		t.Fatal(pyjson.Repr(out), pyjson.Repr(result))
	}
}

func TestAPanickingInverseOrRecoverIsUnverifiedAndInDoubt(t *testing.T) {
	for _, recov := range []bool{false, true} {
		r := newRig(t)
		impl := writeops.SetSlot
		impl.Inverse = func(*plan.Planned, []any) ([]any, error) { panic("boom") }
		if recov {
			impl.Recover = func(plan.Client, *pyjson.Object, *plan.Planned, []any) (any, []any, error) { panic("again") }
		}
		panicked(r, impl)
		_, isErr, out := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
		result := r.lines(t, "journal.jsonl")[1]
		if isErr || get(out, "verdict") != "unverified" || get(out, "readback_error") != "Exception" ||
			get(out, "observed") != nil || pyjson.Repr(get(out, "inverse")) != "[]" || get(result, "in_doubt") != true {
			t.Fatal(recov, pyjson.Repr(out), pyjson.Repr(result))
		}
	}
}
