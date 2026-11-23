package blackbox

import (
	"reflect"
	"strings"
	"testing"
)

// Retro 2026-11-15 (alarm tools and kit module) row 5: a fault is forced on a test point with the
// status flag names, written whole with the value; the hex status string keeps working.

func TestSetSlotStatusTakesFlagNamesAndWritesThemAsTheHexBits(t *testing.T) {
	for _, tc := range []struct{ status, hex string }{
		{"fault", "2"}, {"down,overridden", "24"}, {"{alarm, unackedAlarm}", "88"}, {"null", "40"}, {"ok", "0"},
		{"disabled,fault,down,alarm,stale,overridden,null,unackedAlarm", "ff"},
	} {
		r, c, ord := calcRig(t)
		args := Args{"ord": ord, "slot": "out", "value": 21.5, "value_type": "baja:StatusNumeric", "status": tc.status}
		plan := r.Srv.Dry("n4_set_slot", args)
		if !reflect.DeepEqual(Dig(plan, "plan", "ops"), setOp(c.Handle, "out", StatusNumeric(21.5, tc.hex))) {
			t.Errorf("status %q: ops = %v, want status %q", tc.status, Dig(plan, "plan", "ops"), tc.hex)
		}
		out := r.Srv.RunWrite("n4_set_slot", args)
		if out["verdict"] != "verified" || !reflect.DeepEqual(out["requested"], map[string]any{"value": 21.5, "status": tc.hex}) {
			t.Errorf("status %q: out = %v", tc.status, out)
		}
	}
}

func TestSetSlotStatusKeepsTakingAHexStringAndRefusesAnUnknownName(t *testing.T) {
	r, c, ord := calcRig(t)
	plan := r.Srv.Dry("n4_set_slot", Args{"ord": ord, "slot": "out", "value": 2.0, "value_type": "baja:StatusNumeric", "status": "a"})
	if !reflect.DeepEqual(Dig(plan, "plan", "ops"), setOp(c.Handle, "out", StatusNumeric(2.0, "a"))) {
		t.Errorf("hex status: ops = %v", Dig(plan, "plan", "ops"))
	}
	msg := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": "out", "value": 2.0, "value_type": "baja:StatusNumeric", "status": "faulty"})
	for _, part := range []string{"status must be a hex status string such as '0' or flag names",
		"disabled, fault, down, alarm, stale, overridden, null, unackedAlarm"} {
		if !strings.Contains(msg, part) {
			t.Errorf("refusal lacks %q: %s", part, msg)
		}
	}
}
