package blackbox

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestAddHistoryRefusesEveryFullPolicyButRollOrStopBeforeAnyOp(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	temp := h.Add("3", "Temp", "control:NumericWritable")
	for _, v := range []string{"ROLL", "", "wrap", "Stop", " stop"} {
		text := h.Srv.Err("n4_add_history", covArgs(Args{"full_policy": v}, "Temp"))
		if !strings.Contains(text, "full_policy must be roll or stop, got '"+v+"'") {
			t.Errorf("%q: %q", v, text)
		}
	}
	if hasChild(h.Rig, temp.Handle, "NumericCov") {
		t.Error("a refused call created an extension")
	}
}

func TestAddHistoryRefusesToleranceOnEnumAndStringPointsInAMixedBatch(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Station.SetSupertype("control:EnumWritable", "control:EnumPoint")
	h.Station.SetSupertype("control:EnumPoint", "baja:Component")
	h.Station.SetSupertype("control:StringWritable", "control:StringPoint")
	h.Station.SetSupertype("control:StringPoint", "baja:Component")
	temp := h.Add("3", "Temp", "control:NumericWritable")
	h.Add("3", "Mode", "control:EnumWritable")
	h.Add("3", "Txt", "control:StringWritable")
	for p, family := range map[string]string{"Mode": "an enum", "Txt": "a string"} {
		text := h.Srv.Err("n4_add_history", covArgs(Args{"change_tolerance": 0.5}, "Temp", p))
		if !strings.Contains(text, Folder+"/"+p+" is "+family+" point") {
			t.Errorf("%s: %q", p, text)
		}
	}
	if hasChild(h.Rig, temp.Handle, "NumericCov") {
		t.Error("the numeric point got an extension from a refused batch")
	}
}

func TestAddHistoryFullPolicyStopWithUnlimitedCapacityReadsBack(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	hd := h.Add("3", "Temp", "control:NumericWritable").Handle
	args := historyArgs(Args{"capacity": "unlimited", "full_policy": "stop"}, "Temp")
	notes, _ := Dig(h.Srv.Dry("n4_add_history", args), "plan", "notes").([]any)
	if last, _ := notes[len(notes)-1].(string); !strings.HasPrefix(last, "full_policy stop has no effect with capacity unlimited") {
		t.Errorf("notes = %v", notes)
	}
	out := h.Srv.RunWrite("n4_add_history", args)
	if out["verdict"] != "verified" ||
		nodeValue(h.Rig, hd, "NumericInterval", "historyConfig", "fullPolicy") != "stop" ||
		nodeValue(h.Rig, hd, "NumericInterval", "historyConfig", "capacity") != "0:0" {
		t.Errorf("out = %v", out)
	}
}

// A station failure at the 2nd add stops the batch in-doubt after op 3; n4_rollback refuses
// an in-doubt batch (the docs must not promise otherwise).
func TestAddHistoryFailureAtTheSecondAddIsInDoubtAndRollbackRefuses(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	a := h.Add("3", "AmpsA", "control:NumericWritable")
	b := h.Add("3", "AmpsB", "control:NumericWritable")
	args := covArgs(Args{"change_tolerance": 0.5, "capacity": 50000, "full_policy": "stop"}, "AmpsA", "AmpsB")
	dry := h.Srv.Dry("n4_add_history", args)
	h.Station.SetFaults(boxstation.Faults{Refuse: func(_ *boxstation.Station, key string, arg map[string]any) error {
		for _, op := range objects(arg["ops"]) {
			if key == "syncTo" && op["nm"] == "a" && op["h"] == b.Handle {
				return errors.New("ValueError: boom")
			}
		}
		return nil
	}})
	text := h.Srv.Err("n4_add_history", Execute(args, dry["confirmation_token"]))
	if !strings.Contains(text, "in-doubt") || !strings.Contains(text, "op 4 of 6 failed") ||
		!strings.Contains(text, `Inverse of the sent ops: [{"h":"`+a.Handle+`","n":"NumericCov","nm":"v"}]`) {
		t.Errorf("reply = %q", text)
	}
	if !hasChild(h.Rig, a.Handle, "NumericCov") || hasChild(h.Rig, b.Handle, "NumericCov") {
		t.Fatal("unexpected station state")
	}
	h.Station.SetFaults(boxstation.Faults{})
	lines := h.Lines("journal.jsonl")
	back := h.Srv.Err("n4_rollback", Args{"batch_id": lines[len(lines)-1]["batch_id"]})
	if !strings.Contains(back, "in-doubt") {
		t.Errorf("rollback = %q", back)
	}
}
