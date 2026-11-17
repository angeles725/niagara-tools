package blackbox

// A frozen child's configuration comes back with the rollback (retro 2026-11-16 row M18): the
// rollback of the first remove of a freshly alarmed point lost the alarm algorithm and its
// highLimit/lowLimit. Set ops after the re-add write the captured values of the slots the
// station's type contracts declare as configuration; the plan shows them and the token covers them.

import (
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func double(v string) map[string]any { return map[string]any{"nm": "p", "t": "baja:Double", "v": v} }

func frozenSlot(name, typ, flags string) map[string]any {
	s := map[string]any{"st": "p", "n": name, "v": map[string]any{"nm": "p", "t": typ}}
	if flags != "" {
		s["f"] = flags
	}
	return s
}

// frozenContracts declare a writable's proxyExt, its configuration (tuningPolicyName) and a
// runtime slot (readValue, rt), and an alarm extension's offnormalAlgorithm whose limits are
// configuration.
var frozenContracts = map[string]any{
	"control:NumericWritable": map[string]any{"c": []any{frozenSlot("proxyExt", "control:NullProxyExt", "")}, "p": "baja:Component"},
	"control:BooleanWritable": map[string]any{"c": []any{frozenSlot("proxyExt", "control:NullProxyExt", "")}, "p": "baja:Component"},
	"control:NullProxyExt": map[string]any{"c": []any{frozenSlot("tuningPolicyName", "baja:String", ""),
		frozenSlot("readValue", "baja:String", "rt")}, "p": "baja:Component"},
	"alarm:OutOfRangeAlarmExt": map[string]any{"c": []any{frozenSlot("offnormalAlgorithm", "alarm:OffnormalAlgorithm", "")},
		"p": "baja:Component"},
	"alarm:OutOfRangeAlgorithm": map[string]any{"c": []any{frozenSlot("highLimit", "baja:Double", ""),
		frozenSlot("lowLimit", "baja:Double", "")}, "p": "baja:Component"},
}

func TestARollbackRestoresTheConfigOfAFrozenChild(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(frozenContracts)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	dry := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	var planned map[string]any
	for _, c := range objects(planOf(dry)["components"]) {
		if c["n"] == "proxyExt" && c["parent_path"] == "Sp" {
			planned, _ = c["frozen_config"].(map[string]any)
		}
	}
	if planned == nil || !same(planned["sets"], []any{map[string]any{"n": "tuningPolicyName",
		"b": map[string]any{"nm": "p", "t": "baja:String", "v": "Fast"}}}) {
		t.Fatalf("planned frozen_config = %v (plan %v)", planned, planOf(dry)["components"])
	}
	if len(noteWith(planOf(dry)["notes"], "frozen_config")) != 1 {
		t.Errorf("notes = %v", planOf(dry)["notes"])
	}
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if _, ok := back["frozen_config_not_restored"]; ok {
		t.Errorf("frozen_config_not_restored = %v", back["frozen_config_not_restored"])
	}
	if got := d.value(d.node(nn, "Sp", "proxyExt").Handle, "tuningPolicyName"); got != "Fast" {
		t.Errorf("restored tuningPolicyName = %v, want Fast", got)
	}
}

// A runtime slot (declared rt) is never written, and its value is not compared.
func TestARollbackNeverWritesARuntimeSlotOfAFrozenChild(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(frozenContracts)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	d.Set(d.node(nn, "Sp", "proxyExt").Handle, "readValue", map[string]any{"nm": "p", "t": "baja:String", "v": "42"})
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	back := d.rollback(removed["batch_id"])
	for _, op := range sent() {
		if n, _ := op["n"].(string); strings.HasSuffix(n, "readValue") {
			t.Errorf("a runtime slot was written: %v", op)
		}
	}
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
}

// A value the station refuses stays in frozen_config_not_restored, with the station's reason.
func TestAFrozenConfigTheStationRefusesStaysNotRestored(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(frozenContracts)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	d.f.set(func(f *faults) {
		f.replace = func(sck string, arg map[string]any) (any, error, bool) {
			for _, op := range objects(arg["ops"]) {
				if n, _ := op["n"].(string); sck == "syncTo" && strings.HasSuffix(n, "tuningPolicyName") {
					return nil, errors.New("Cannot set tuningPolicyName"), true
				}
			}
			return nil, nil, false
		}
	})
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "partial" {
		t.Errorf("back = %v", back)
	}
	lost := objects(back["frozen_config_not_restored"])
	if len(lost) != 1 || !same(lost[0]["slots"], []any{"tuningPolicyName"}) {
		t.Fatalf("frozen_config_not_restored = %v", lost)
	}
	refused := objects(lost[0]["refused"])
	if len(refused) != 1 || refused[0]["slot"] != "tuningPolicyName" ||
		!strings.Contains(refused[0]["error"].(string), "Cannot set tuningPolicyName") {
		t.Errorf("refused = %v", lost[0]["refused"])
	}
}

// The M18 shape: a fresh alarm extension holds its default algorithm; the rollback swaps the
// captured algorithm type in, then writes its limits.
func TestARollbackRestoresAnAlarmAlgorithmAndItsLimits(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(frozenContracts)
	d.afterAdd(func(st *boxstation.Station, n *boxstation.Node) {
		if n.Type == "alarm:OutOfRangeAlarmExt" && n.Child("offnormalAlgorithm") == nil {
			st.Component(n, "offnormalAlgorithm", "alarm:OffnormalAlgorithm", "")
		}
	})
	amps := d.Add("3", "Amps", "baja:Folder")
	alarm := d.Add(amps.Handle, "Alarm", "alarm:OutOfRangeAlarmExt")
	d.Set(alarm.Handle, "offnormalAlgorithm", map[string]any{"nm": "p", "t": "alarm:OutOfRangeAlgorithm", "v": ""})
	d.Set(alarm.Handle, "offnormalAlgorithm/highLimit", double("80.0"))
	d.Set(alarm.Handle, "offnormalAlgorithm/lowLimit", double("10.0"))
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(amps.Name))
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	algo := d.node(amps.Name, "Alarm", "offnormalAlgorithm")
	if algo == nil || algo.Type != "alarm:OutOfRangeAlgorithm" {
		t.Fatalf("offnormalAlgorithm = %+v", algo)
	}
	for slot, want := range map[string]string{"highLimit": "80.0", "lowLimit": "10.0"} {
		if got := d.value(d.node(amps.Name, "Alarm").Handle, "offnormalAlgorithm", slot); got != want {
			t.Errorf("restored %s = %v, want %s", slot, got, want)
		}
	}
}

// A readonly (r, persisted) slot is never written, but it is still compared: a difference keeps
// the frozen child in frozen_config_not_restored and the verdict partial (verifier probe P3).
func TestAReadonlyFrozenConfigDifferenceIsNotWrittenAndStaysNotRestored(t *testing.T) {
	d := destructiveRig(t, nil)
	contracts := map[string]any{}
	for k, v := range frozenContracts {
		contracts[k] = v
	}
	contracts["control:NullProxyExt"] = map[string]any{"c": []any{frozenSlot("tuningPolicyName", "baja:String", "r")},
		"p": "baja:Component"}
	d.withContractsOf(contracts)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	back := d.rollback(removed["batch_id"])
	for _, op := range sent() {
		if n, _ := op["n"].(string); strings.HasSuffix(n, "tuningPolicyName") {
			t.Errorf("a readonly slot was written: %v", op)
		}
	}
	if back["verdict"] != "partial" {
		t.Errorf("verdict = %v", back["verdict"])
	}
	lost := objects(back["frozen_config_not_restored"])
	if len(lost) != 1 || !same(lost[0]["slots"], []any{"tuningPolicyName"}) {
		t.Errorf("frozen_config_not_restored = %v", back["frozen_config_not_restored"])
	}
}
