package plan_test

import (
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// budgeted gives the rig's set_slot a reply budget and the server's background (#925).
func budgeted(r *rig, budget time.Duration) {
	e := &plan.Engine{Ctx: r.ctx, State: r.state, ReplyBudget: budget, Background: r.srv.Background}
	r.srv.Handlers["n4_set_slot"] = e.Handler("n4_set_slot", writeops.SetSlot)
}

// A write still running when its reply budget runs out answers {state: running, batch_id} at
// once; it goes on in the background, journals its result under that batch and audits its end.
func TestAWritePastItsReplyBudgetAnswersRunning(t *testing.T) {
	r := newRig(t)
	budgeted(r, 20*time.Millisecond)
	tok := r.dry(t, "n4_set_slot", setArgs())
	r.st.onSend = func() { time.Sleep(300 * time.Millisecond) }
	start := time.Now()
	text, isErr, out := r.execute(t, "n4_set_slot", setArgs(), tok)
	if took := time.Since(start); isErr || took > 250*time.Millisecond || get(out, "state") != "running" {
		t.Fatalf("%v %s %s", took, text, pyjson.Repr(out))
	}
	batch := get(out, "batch_id").(string)
	if audit := r.lines(t, "audit.jsonl"); get(audit[len(audit)-1], "outcome") != "running" ||
		get(audit[len(audit)-1], "batch_id") != batch {
		t.Fatal(pyjson.Repr(audit))
	}
	r.srv.Background.Wait()
	journal, audit := r.lines(t, "journal.jsonl"), r.lines(t, "audit.jsonl")
	last := journal[len(journal)-1]
	if get(last, "batch_id") != batch || get(last, "phase") != "result" || get(last, "in_doubt") == true {
		t.Fatal(pyjson.Repr(journal))
	}
	if end := audit[len(audit)-1]; get(end, "outcome") != "executed" || get(end, "batch_id") != batch {
		t.Fatal(pyjson.Repr(audit))
	}
}

// A write that ends inside its budget answers as before: the whole result, no state.
func TestAWriteInsideItsReplyBudgetAnswersItsResult(t *testing.T) {
	r := newRig(t)
	budgeted(r, 5*time.Second)
	tok := r.dry(t, "n4_set_slot", setArgs())
	text, isErr, out := r.execute(t, "n4_set_slot", setArgs(), tok)
	if _, has := out.Get("state"); isErr || has || get(out, "verdict") == nil {
		t.Fatalf("%s %s", text, pyjson.Repr(out))
	}
}
