package blackbox

// n4_add_history: a numeric change-of-value extension's change tolerance and the history's full
// policy, for N points under one dry run and one token (client-A overamps retro 2026-11-16,
// row M4: three COV histories with a 0.5 A tolerance and 50000 records took 30 calls).

import (
	"strings"
	"testing"
)

// covArgs are cov-mode n4_add_history arguments for points, plus extra (no interval_ms).
func covArgs(extra Args, points ...string) Args {
	args := historyArgs(withArgs(Args{"mode": "cov"}, extra), points...)
	delete(args, "interval_ms")
	return args
}

func TestAddHistoryCovToleranceCapacityAndFullPolicyForThreePointsInOneToken(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	names := []string{"AmpsA", "AmpsB", "AmpsC"}
	handles := []string{}
	for _, n := range names {
		handles = append(handles, h.Add("3", n, "control:NumericWritable").Handle)
	}
	args := covArgs(Args{"change_tolerance": 0.5, "capacity": 50000, "full_policy": "stop"}, names...)
	dry := h.Srv.Dry("n4_add_history", args)
	if ops, _ := Dig(dry, "plan", "ops").([]any); len(ops) != 9 {
		t.Fatalf("one dry run plans 3 adds and 6 sets, got %d ops: %v", len(ops), dry)
	}
	out := h.Srv.OK("n4_add_history", Execute(args, dry["confirmation_token"]))
	observed, _ := out["observed"].([]any)
	if out["verdict"] != "verified" || len(observed) != 3 {
		t.Fatalf("out = %v", out)
	}
	for i, hd := range handles {
		if got := Dig(observed[i], "changeTolerance"); got != 0.5 {
			t.Errorf("%s observed changeTolerance = %v", names[i], got)
		}
		if got := Dig(observed[i], "fullPolicy"); got != "stop" {
			t.Errorf("%s observed fullPolicy = %v", names[i], got)
		}
		if got := nodeValue(h.Rig, hd, "NumericCov", "changeTolerance"); got != "0.5" {
			t.Errorf("%s changeTolerance on the station = %v", names[i], got)
		}
		if got := nodeValue(h.Rig, hd, "NumericCov", "historyConfig", "capacity"); got != "1:50000" {
			t.Errorf("%s capacity on the station = %v", names[i], got)
		}
		if got := nodeValue(h.Rig, hd, "NumericCov", "historyConfig", "fullPolicy"); got != "stop" {
			t.Errorf("%s fullPolicy on the station = %v", names[i], got)
		}
	}
}

func TestAddHistoryFullPolicyStopIsInTheCapacityNote(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	dry := h.Srv.Dry("n4_add_history", historyArgs(Args{"capacity": 2016, "full_policy": "stop"}, "Temp"))
	notes, _ := Dig(dry, "plan", "notes").([]any)
	if len(notes) < 2 || !strings.HasPrefix(notes[1].(string), "Capacity 2016 records (Stop)") {
		t.Errorf("notes = %v", notes)
	}
}

func TestAddHistoryRefusesAChangeToleranceThatDoesNotApply(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	temp := h.Add("3", "Temp", "control:NumericWritable")
	cases := []struct {
		args Args
		want string
	}{
		{historyArgs(Args{"change_tolerance": 0.5}, "Temp"), "change_tolerance applies only to mode cov"},
		{covArgs(Args{"change_tolerance": 0.5}, "Temp", "Fan"),
			"change_tolerance applies only to a numeric point's NumericCov extension: " + Folder + "/Fan is a boolean point"},
		{covArgs(Args{"change_tolerance": 0.5, "ext_type": "history:BooleanCovHistoryExt"}, "Fan"),
			"change_tolerance applies only to history:NumericCovHistoryExt"},
	}
	for _, c := range cases {
		if text := h.Srv.Err("n4_add_history", c.args); !strings.Contains(text, c.want) {
			t.Errorf("%v: refusal = %q, want %q", c.args, text, c.want)
		}
	}
	for _, bad := range []any{-1, "0.5", true} { // the schema: a number, minimum 0
		if code := h.Srv.CallError("n4_add_history", covArgs(Args{"change_tolerance": bad}, "Temp")); code != -32602 {
			t.Errorf("change_tolerance %v: code = %d, want -32602", bad, code)
		}
	}
	if hasChild(h.Rig, temp.Handle, "NumericCov") || hasChild(h.Rig, fan.Handle, "BooleanCov") {
		t.Error("a refused call created an extension")
	}
}
