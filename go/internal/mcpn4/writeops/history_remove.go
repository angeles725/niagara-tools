package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// RemoveHistory is n4_remove_history: one deleteHistories call on the station's history channel
// (box-rt BHistoryChannel.deleteHistories: body [ORD], each "history:<id>" as the station's own
// Fox client builds it, FoxHistorySpaceConnection.deleteHistory). The engine only requires some
// write scope; the plan maps the history to the extension that records into it and checks that
// extension's slot path against the scope. No inverse: the records cannot be brought back.
var RemoveHistory = plan.Impl{ScopeOrds: func(*pyjson.Object) []any { return nil }, PlanWith: removeHistoryPlan,
	Readback: removeHistoryReadback, Unapplied: removeHistoryUnapplied}

// The history channel call.
const (
	historyChannel = "history"
	historyDelete  = "deleteHistories"
)

// removeHistoryPlan checks the acknowledgement, the id, the client, and the extensions that
// record into the history (in scope, none enabled unless allowed); the plan is the one channel op.
func removeHistoryPlan(c plan.Client, args *pyjson.Object, w *plan.State, _ *session.Session) (*plan.Planned, error) {
	if get(args, "acknowledge_data_loss") != true {
		return nil, hrRefused(msgHRAck)
	}
	id := str(args, "history_id")
	if err := bql.ValidateHistoryId(id); err != nil {
		return nil, plan.ToolError(err.Error())
	}
	r, ok := c.(ordReader)
	if !ok || !plan.CanCall(c) {
		return nil, hrRefused(msgHRNoCall)
	}
	exts, err := readHistoryExts(r, id)
	if err != nil {
		return nil, err
	}
	allowed := get(args, "allow_enabled_extension") == true
	notes := []any{noteHRNoInverse}
	shown := []any{}
	for _, e := range exts {
		if w.Scope.Check(e.ord) != nil {
			return nil, hrRefused(fmt.Sprintf(msgHROutside, e.ord))
		}
		if e.enabled && !allowed {
			return nil, hrRefused(fmt.Sprintf(msgHREnabled, e.ord))
		}
		notes = append(notes, fmt.Sprintf(noteHRExt, e.ord, e.enabled))
		if e.enabled {
			notes = append(notes, fmt.Sprintf(noteHREnabledExt, e.ord))
		}
		shown = append(shown, obj("ext_ord", e.ord, "enabled", e.enabled))
	}
	_, count, note, err := readHistoryCount(r, id)
	if err != nil {
		return nil, err
	}
	info := obj("history_id", id, "record_count", count, "extensions", shown)
	if note != "" {
		info.Set("count_note", note)
	}
	return &plan.Planned{Ops: []any{plan.ChannelOp(historyChannel, historyDelete, []any{bql.HistoryScheme + id})},
		Notes: notes, Data: id, Preview: obj("history_removal", info)}, nil
}

func hrRefused(msg string) error { return plan.ToolError(fmt.Sprintf(msgHRRefused, msg)) }
