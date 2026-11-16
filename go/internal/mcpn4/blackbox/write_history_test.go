package blackbox

// n4_add_history: the extension type follows the point type, a repeated history id is refused
// at the dry run, the capacity note follows the interval and the read-back checks the history id
// and fault (bench retro 2026-11-15 row 7).

import (
	"strings"
	"testing"
)

func TestAddHistoryOnABooleanPointCreatesABooleanInterval(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	temp := h.Add("3", "Temp", "control:NumericWritable")
	out := h.Srv.RunWrite("n4_add_history", historyArgs(nil, "Fan", "Temp"))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	if got := historyOf(h.Rig, fan.Handle, "BooleanInterval"); got != "history:BooleanIntervalHistoryExt" {
		t.Errorf("Fan/BooleanInterval type = %q", got)
	}
	if got := historyOf(h.Rig, temp.Handle, "NumericInterval"); got != "history:NumericIntervalHistoryExt" {
		t.Errorf("Temp/NumericInterval type = %q", got)
	}
}

func TestAddHistoryRefusesAMismatchedTypeAtTheDryRun(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	text := h.Srv.Err("n4_add_history", historyArgs(Args{"ext_type": "history:NumericIntervalHistoryExt"}, "Fan"))
	if !strings.Contains(text, "a boolean point (control:BooleanWritable): use history:BooleanIntervalHistoryExt") {
		t.Errorf("refusal = %q", text)
	}
	if hasChild(h.Rig, fan.Handle, "NumericInterval") {
		t.Error("an extension was created")
	}
}

func TestAddHistoryRefusesARepeatedHistoryId(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs+"slot:/Folder/Old/NumericInterval,/FakeStation/Temp\n")
	h.Add("3", "Temp", "control:NumericWritable")
	text := h.Srv.Err("n4_add_history", historyArgs(nil, "Temp"))
	if !strings.Contains(text, "history id '/FakeStation/Temp' of "+Folder+"/Temp is already used by the "+
		"extension at slot:/Folder/Old/NumericInterval") {
		t.Errorf("station repeat: %q", text)
	}
	a, b := h.Add("3", "A", "baja:Folder"), h.Add("3", "B", "baja:Folder")
	h.Add(a.Handle, "Temp", "control:NumericWritable")
	h.Add(b.Handle, "Temp", "control:NumericWritable")
	text = h.Srv.Err("n4_add_history", historyArgs(nil, "A/Temp", "B/Temp"))
	if !strings.Contains(text, "history name 'Temp' would repeat on "+Folder+"/A/Temp and "+Folder+"/B/Temp") {
		t.Errorf("batch repeat: %q", text)
	}
}

func TestAddHistoryCapacityNoteUsesTheRealInterval(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	dry := h.Srv.Dry("n4_add_history", historyArgs(Args{"interval_ms": 100}, "Temp"))
	notes, _ := Dig(dry, "plan", "notes").([]any)
	want := "Capacity stays at the extension default (500 records, Roll): about 50 s at 100 ms."
	if len(notes) < 2 || notes[1] != want {
		t.Errorf("notes = %v, want %q", notes, want)
	}
}

func TestAddHistoryInFaultReadsMismatch(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.faultCause = "Duplicate history id: /FakeStation/Temp"
	h.Add("3", "Temp", "control:NumericWritable")
	out := h.Srv.RunWrite("n4_add_history", historyArgs(nil, "Temp"))
	observed, _ := out["observed"].([]any)
	if out["verdict"] != "mismatch" || len(observed) != 1 ||
		Dig(observed[0].(map[string]any), "faultCause") != h.faultCause ||
		Dig(observed[0].(map[string]any), "historyId") != "/FakeStation/Temp" {
		t.Errorf("out = %v", out)
	}
}
