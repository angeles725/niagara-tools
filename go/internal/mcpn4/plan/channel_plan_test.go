package plan_test

// A plan holding a channel op outside the allow-list is refused at plan time, before any
// token, journal record or station call: "nothing was planned".

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestADisallowedChannelOpIsRefusedAtPlanTime(t *testing.T) {
	r := newRig(t)
	impl := plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return []any{calc} },
		Plan: func(plan.Client, *pyjson.Object) (*plan.Planned, error) {
			return &plan.Planned{Ops: []any{plan.ChannelOp("alarm", "clearAllRecords", node())}}, nil
		}}
	r.srv.Handlers["n4_invoke_action"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_invoke_action", impl)
	for _, args := range []*pyjson.Object{node("ord", calc, "action", "x"), node("ord", calc, "action", "x", "dry_run", false, "confirmation_token", "x")} {
		text, isErr, _ := r.call(t, "n4_invoke_action", args)
		if !isErr || !strings.Contains(text, `channel "alarm" key "clearAllRecords" is not an allowed channel op`) ||
			!strings.Contains(text, "nothing was planned") {
			t.Errorf("%s: %v %s", pyjson.Repr(args), isErr, text)
		}
	}
	if len(r.st.sent) != 0 || r.lines(t, "journal.jsonl") != nil || r.state.WritesExecuted != 0 {
		t.Errorf("sent %v, journal %v", r.st.sent, r.lines(t, "journal.jsonl"))
	}
}
