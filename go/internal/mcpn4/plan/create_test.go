package plan_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Go twin of test_tools_destructive.TestPlannedDataGuards (go_twin("plan", "injection")): a plan
// with no private data and no targets (Python: data that is not a dict) still executes, and its
// intent records empty targets.
func TestAPlanWhoseDataIsNotADictStillExecutes(t *testing.T) {
	r := newRig(t)
	impl := writeops.CreateComponent
	plan0 := impl.Plan
	impl.Plan = func(c plan.Client, args *pyjson.Object) (*plan.Planned, error) {
		p, err := plan0(c, args)
		if err == nil {
			p.Data, p.Targets = nil, nil
		}
		return p, err
	}
	impl.Inverse = func(*plan.Planned, []any) ([]any, error) { return []any{}, nil }
	impl.Readback = func(_ plan.Client, _ *pyjson.Object, _ *plan.Planned, replies, _ []any) (any, any, any, string, error) {
		return node(), replies[0], node(), "verified", nil
	}
	r.srv.Handlers["n4_create_component"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_create_component", impl)
	args := node("parent_ord", calc, "name", "Pump", "type", "kitControl:NumericConst")
	text, isErr, out := r.execute(t, "n4_create_component", args, r.dry(t, "n4_create_component", args))
	intent := r.lines(t, "journal.jsonl")[0]
	if isErr || get(out, "verdict") != "verified" || pyjson.Repr(get(intent, "targets")) != "{}" || len(r.st.sent) != 1 {
		t.Fatal(text, pyjson.Repr(out), pyjson.Repr(intent))
	}
}
