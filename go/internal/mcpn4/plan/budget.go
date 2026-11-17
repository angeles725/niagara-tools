package plan

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// saveTool persists what the other writes changed. The write budget (--max-writes) bounds changes,
// never the save: a bulk job that spent the budget must still be able to persist its work (field
// retro 2026-11-16, row U7), so the save neither needs nor spends a write.
const saveTool = "n4_save_station"

// spendsBudget says whether executing the tool counts against --max-writes.
func spendsBudget(name string) bool { return name != saveTool }

// budgetRefusal is the refusal of a write once the process budget is spent.
func budgetRefusal(max int) error {
	return ToolError(fmt.Sprintf("%s: %d writes already executed by this server process "+
		"(--max-writes; a reconnect does not reset it); n4_save_station still runs, so save the work "+
		"done so far, then restart the server for a new budget", safety.ReasonBudget, max))
}

// budgetReport is the write_budget object every write reply carries, so a bulk job sees how many
// writes it has left before it plans the next one.
func (w *State) budgetReport() any {
	left := w.MaxWrites - w.WritesExecuted
	if left < 0 {
		left = 0
	}
	return obj("executed", w.WritesExecuted, "max", w.MaxWrites, "remaining", left, "save_counts", false)
}

// cost is what executing the batch spends of the budget: Writes, at least one.
func (p *Planned) cost() int { return max(p.Writes, 1) }

// CheckRoom refuses a plan of need writes that the remaining budget cannot hold, before its dry
// run issues a token (field retro 2026-11-16, row U7): what refuses is the plan, never a batch
// already half sent. what names the batch in the refusal ("this replication").
func (w *State) CheckRoom(need int, what string) error {
	left := max(w.MaxWrites-w.WritesExecuted, 0)
	if need <= left {
		return nil
	}
	return ToolError(fmt.Sprintf("%s: %s needs %d write(s) but only %d remain (--max-writes %d; "+
		"n4_save_station is outside the budget): split it, or save and restart the server for a new budget",
		safety.ReasonBudgetSmall, what, need, left, w.MaxWrites))
}
