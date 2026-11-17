package blackbox

import (
	"reflect"
	"strings"
	"testing"
)

// The write budget (--max-writes) bounds changes, never the save that persists them: a bulk job that
// spent the budget must still be able to save (field retro 2026-11-16, row U7: a replication stopped at
// the budget left the station unsaved).

func TestSaveRunsAfterTheBudgetIsSpent(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	out := r.Srv.RunWrite("n4_save_station", nil)
	if _, ok := out["verdict"]; !ok {
		t.Errorf("save after a spent budget = %v", out)
	}
}

func TestTheSaveDoesNotSpendTheBudget(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_save_station", nil)
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"}))
	if !strings.Contains(text, "write budget exhausted") || !reflect.DeepEqual(r.Children(), []string{"A"}) {
		t.Errorf("refusal = %s, children %v", text, r.Children())
	}
}

func TestEveryWriteReplyReportsTheBudget(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 3 })
	r.ConnectVerified()
	dry := r.Srv.Dry("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	want := map[string]any{"executed": float64(0), "max": float64(3), "remaining": float64(3), "save_counts": false, "rollback_counts": false}
	if !same(dry["write_budget"], want) {
		t.Errorf("dry run write_budget = %v, want %v", dry["write_budget"], want)
	}
	out := r.Srv.OK("n4_create_component", Execute(withArgs(createPump, Args{"name": "A"}), dry["confirmation_token"]))
	want = map[string]any{"executed": float64(1), "max": float64(3), "remaining": float64(2), "save_counts": false, "rollback_counts": false}
	if !same(out["write_budget"], want) {
		t.Errorf("execute write_budget = %v, want %v", out["write_budget"], want)
	}
}

func TestTheBudgetRefusalSaysTheSaveIsStillAllowed(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"}))
	if !strings.Contains(text, "n4_save_station and n4_rollback still run") {
		t.Errorf("refusal = %s", text)
	}
}

// Recovery stays reachable after the budget is spent (retro 2026-11-17 mcpn4-replicate-points Δ6):
// every n4_rollback is outside --max-writes, like the save, and spends nothing; the next change is
// still refused.
func TestTheSaveAndTheRollbackRunAfterTheBudgetIsSpent(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	created := r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": created["batch_id"]})
	if back["verdict"] != "verified" || len(r.Children()) != 0 {
		t.Errorf("rollback after a spent budget = %v, children %v", back, r.Children())
	}
	want := map[string]any{"executed": float64(1), "max": float64(1), "remaining": float64(0), "save_counts": false,
		"rollback_counts": false}
	if !same(back["write_budget"], want) {
		t.Errorf("write_budget after the rollback = %v, want %v", back["write_budget"], want)
	}
	if out := r.Srv.RunWrite("n4_save_station", nil); out["verdict"] == nil {
		t.Errorf("save after a spent budget = %v", out)
	}
	if text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"})); !strings.Contains(text, "write budget exhausted") {
		t.Errorf("a change after the rollback = %s", text)
	}
}
