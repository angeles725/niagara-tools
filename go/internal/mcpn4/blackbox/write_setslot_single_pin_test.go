package blackbox

// n4_set_slot single form after the batch form (OD040): its plan and reply keep their shape.

import (
	"reflect"
	"testing"
)

// The single form keeps its reply: the same keys, values and plan as before the batch form.
func TestTheSingleFormReplyIsUnchanged(t *testing.T) {
	r, a, _ := batchRig(t)
	args := Args{"ord": Folder + "/A", "slot": "out", "value": 5.5, "value_type": "baja:StatusNumeric"}
	plan := r.Srv.Dry("n4_set_slot", args)
	if got := keys(plan["plan"].(map[string]any)); !reflect.DeepEqual(got, []string{"inverse", "notes", "ops", "station", "tool"}) {
		t.Errorf("plan keys = %v", got)
	}
	out := r.Srv.OK("n4_set_slot", Execute(args, plan["confirmation_token"]))
	wantKeys := []string{"accepted", "batch_id", "dry_run", "inverse", "observed", "requested", "verdict", "write_budget"}
	if got := keys(out); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("keys = %v", got)
	}
	want := map[string]any{"value": 5.5, "status": "0"}
	if !reflect.DeepEqual(out["requested"], want) || !reflect.DeepEqual(out["observed"], want) ||
		!reflect.DeepEqual(out["inverse"], setOp(a.Handle, "out", StatusNumeric(1.0, "0"))) {
		t.Errorf("out = %v", out)
	}
	if text := r.Srv.Err("n4_set_slot", Args{"slot": "out", "value": 1.0, "value_type": "baja:Double"}); text != "missing required argument: ord" {
		t.Errorf("missing ord: %q", text)
	}
}
