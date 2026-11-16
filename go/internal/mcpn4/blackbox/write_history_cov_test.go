package blackbox

// n4_add_history: change-of-value extensions and the capacity argument (bench retro 2026-11-15
// row 8, client-A history rollout retro row 2).

import (
	"strings"
	"testing"
)

func TestAddHistoryCovModeCreatesAChangeOfValueExtension(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	args := historyArgs(Args{"mode": "cov"}, "Fan")
	delete(args, "interval_ms")
	out := h.Srv.RunWrite("n4_add_history", args)
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	if got := historyOf(h.Rig, fan.Handle, "BooleanCov"); got != "history:BooleanCovHistoryExt" {
		t.Errorf("Fan/BooleanCov type = %q", got)
	}
	if got := nodeValue(h.Rig, fan.Handle, "BooleanCov", "interval"); got != "<absent>" {
		t.Errorf("a COV extension got an interval: %v", got)
	}
	if text := h.Srv.Err("n4_add_history", historyArgs(Args{"mode": "cov"}, "Fan")); !strings.Contains(text,
		"interval_ms does not apply to mode cov") {
		t.Errorf("cov with an interval: %q", text)
	}
}

func TestAddHistoryCapacityIsSetAndReadBack(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	temp := h.Add("3", "Temp", "control:NumericWritable")
	out := h.Srv.RunWrite("n4_add_history", historyArgs(Args{"capacity": 20160}, "Temp"))
	observed, _ := out["observed"].([]any)
	if out["verdict"] != "verified" || len(observed) != 1 || Dig(observed[0], "capacity") != "1:20160" {
		t.Fatalf("out = %v", out)
	}
	if got := nodeValue(h.Rig, temp.Handle, "NumericInterval", "historyConfig", "capacity"); got != "1:20160" {
		t.Errorf("capacity on the station = %v", got)
	}
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	args := historyArgs(Args{"mode": "cov", "capacity": "unlimited"}, "Fan")
	delete(args, "interval_ms")
	dry := h.Srv.Dry("n4_add_history", args)
	if notes, _ := Dig(dry, "plan", "notes").([]any); len(notes) < 2 ||
		!strings.HasPrefix(notes[1].(string), "Capacity unlimited") {
		t.Errorf("unlimited notes = %v", notes)
	}
	h.Srv.OK("n4_add_history", Execute(args, dry["confirmation_token"]))
	if got := nodeValue(h.Rig, fan.Handle, "BooleanCov", "historyConfig", "capacity"); got != "0:0" {
		t.Errorf("unlimited capacity on the station = %v", got)
	}
	for _, bad := range []any{0, -1, 2147483648, "500 records", 1.5} {
		if text := h.Srv.Err("n4_add_history", historyArgs(Args{"capacity": bad}, "Temp")); !strings.Contains(text,
			"capacity must be a whole number of records") {
			t.Errorf("capacity %v: %q", bad, text)
		}
	}
}
