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
	want := map[string]any{"executed": float64(0), "max": float64(3), "remaining": float64(3), "save_counts": false}
	if !same(dry["write_budget"], want) {
		t.Errorf("dry run write_budget = %v, want %v", dry["write_budget"], want)
	}
	out := r.Srv.OK("n4_create_component", Execute(withArgs(createPump, Args{"name": "A"}), dry["confirmation_token"]))
	want = map[string]any{"executed": float64(1), "max": float64(3), "remaining": float64(2), "save_counts": false}
	if !same(out["write_budget"], want) {
		t.Errorf("execute write_budget = %v, want %v", out["write_budget"], want)
	}
}

func TestTheBudgetRefusalSaysTheSaveIsStillAllowed(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.MaxWrites = 1 })
	r.ConnectVerified()
	r.Srv.RunWrite("n4_create_component", withArgs(createPump, Args{"name": "A"}))
	text := r.Srv.Err("n4_create_component", withArgs(createPump, Args{"name": "B"}))
	if !strings.Contains(text, "n4_save_station still runs") {
		t.Errorf("refusal = %s", text)
	}
}
