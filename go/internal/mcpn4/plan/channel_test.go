package plan_test

// The channel allow-list is data: exactly the audited ops, and the history channel's other keys
// (clearAllRecords, clearOldRecords) stay refused at plan time.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestTheChannelAllowListIsExactlyTheAuditedOps(t *testing.T) {
	want := map[string][]string{"alarm": {"ackAlarms"}, "history": {"deleteHistories"}}
	if !reflect.DeepEqual(plan.ChannelAllowList, want) {
		t.Errorf("allow-list %v, want %v", plan.ChannelAllowList, want)
	}
}

func TestTheOtherHistoryChannelKeysAreRefusedAtPlanTime(t *testing.T) {
	for _, key := range []string{"clearAllRecords", "clearOldRecords", "renameHistory"} {
		r := newRig(t)
		impl := plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return []any{calc} },
			Plan: func(plan.Client, *pyjson.Object) (*plan.Planned, error) {
				return &plan.Planned{Ops: []any{plan.ChannelOp("history", key, []any{"history:/S/H"})}}, nil
			}}
		r.srv.Handlers["n4_invoke_action"] = (&plan.Engine{Ctx: r.ctx, State: r.state}).Handler("n4_invoke_action", impl)
		text, isErr, _ := r.call(t, "n4_invoke_action", node("ord", calc, "action", "x"))
		if !isErr || !strings.Contains(text, `channel "history" key "`+key+`" is not an allowed channel op`) ||
			!strings.Contains(text, "nothing was planned") {
			t.Errorf("%s: %v %s", key, isErr, text)
		}
		if len(r.st.sent) != 0 {
			t.Errorf("%s: sent %v", key, r.st.sent)
		}
	}
}
