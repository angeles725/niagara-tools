package plan_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// failingResult refuses the result records, as an OSError would.
type failingResult struct{ *journal.Journal }

func (f failingResult) Append(e *pyjson.Object) error {
	if get(e, "phase") == "result" {
		return errors.New("disk full")
	}
	return f.Journal.Append(e)
}

// twice sends n4_set_slot's op twice; the second send fails.
func twice(r *rig) {
	impl := writeops.SetSlot
	setPlan := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := setPlan(c, args)
		if err == nil {
			p.Ops = []any{p.Ops[0], p.Ops[0]}
		}
		return p, err
	}
	panicked(r, impl)
	r.st.onSend = func() { r.st.sendErr = errors.New("boom") }
}

// _in_doubt_partial: op 2 of 2 fails after op 1 was accepted. The error names the sent ops,
// their replies and inverse, and an in-doubt result record keeps the batch in-doubt.
func TestAMultiOpBatchThatFailsMidWayNamesTheSentOpsAndTheirInverse(t *testing.T) {
	r := newRig(t)
	twice(r)
	text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	lines := r.lines(t, "journal.jsonl")
	last := lines[len(lines)-1]
	if !isErr || !strings.HasSuffix(text, ": Exception; op 2 of 2 failed: op 1 was sent and accepted by the station "+
		"(replies: [[]]), op 2 may or may not have been applied and no later op was sent. Inverse of the sent ops: "+
		`[{"b":{"nm":"p","s":[{"n":"value","nm":"p","t":"baja:Double","v":"1.0"},{"n":"status","nm":"p","t":"baja:Status","v":"0"}],"t":"baja:StatusNumeric"},"h":"10","n":"out","nm":"s"}]`) ||
		get(last, "in_doubt") != true || pyjson.Repr(get(last, "sent")) != "1" || pyjson.Repr(get(last, "failed_op")) != "1" {
		t.Fatalf("%s\n%s", text, pyjson.Repr(last))
	}
}

func TestAPartialResultThatCannotBeJournaledIsSaidAndStaysInDoubt(t *testing.T) {
	r := newRig(t)
	twice(r)
	tok := r.dry(t, "n4_set_slot", setArgs())
	r.state.Journal = failingResult{journal.New(r.dir)}
	text, _, _ := r.execute(t, "n4_set_slot", setArgs(), tok)
	if !strings.Contains(text, "op 2 of 2 failed") || !strings.HasSuffix(text, " (the in-doubt result record could not be journaled)") {
		t.Fatal(text)
	}
}

// Any failure after the intent, not only a station call, is the batch's in-doubt error: in
// the reply and the audit line, with its batch id.
func TestAnUnexpectedFailureAfterTheIntentKeepsTheBatchID(t *testing.T) {
	r := newRig(t)
	impl := writeops.SetSlot
	impl.Run = func(*plan.Batch, *plan.Planned, []any) (*plan.Runs, error) {
		return nil, errors.New("not JSON serializable")
	}
	panicked(r, impl)
	text, isErr, _ := r.execute(t, "n4_set_slot", setArgs(), r.dry(t, "n4_set_slot", setArgs()))
	batch := get(r.lines(t, "journal.jsonl")[0], "batch_id").(string)
	audit := r.lines(t, "audit.jsonl")
	if !isErr || !strings.Contains(text, "batch "+batch+" is in-doubt") || !strings.HasSuffix(text, ": Exception") ||
		get(audit[1], "outcome") != "error" || get(audit[1], "batch_id") != batch {
		t.Fatalf("%s %s", text, pyjson.Repr(audit[1]))
	}
}
