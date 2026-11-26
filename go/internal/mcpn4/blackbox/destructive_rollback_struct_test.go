package blackbox

// B1200-G3 (issue #200): a rollback re-creates only components. alarm:LimitEnable is a baja:Struct
// by the station registry, a slot value the naming rule (every type outside baja) took for a
// component, so the rollback split it out, reported it missing and left the alarm disabled.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// structContracts add to frozenContracts the way a station describes an out-of-range algorithm:
// its limitEnable is a frozen alarm:LimitEnable, a baja:Struct.
func structContracts() map[string]any {
	out := map[string]any{}
	for k, v := range frozenContracts {
		out[k] = v
	}
	out["alarm:OutOfRangeAlgorithm"] = map[string]any{"c": []any{frozenSlot("highLimit", "baja:Double", ""),
		frozenSlot("lowLimit", "baja:Double", ""), frozenSlot("limitEnable", "alarm:LimitEnable", "")}, "p": "baja:Component"}
	out["alarm:LimitEnable"] = map[string]any{"c": []any{frozenSlot("highLimitEnable", "baja:Boolean", ""),
		frozenSlot("lowLimitEnable", "baja:Boolean", "")}, "p": "baja:Struct"}
	out["baja:Struct"] = map[string]any{}
	return out
}

// alarmedAmps is a Folder with an out-of-range alarm extension: highLimit 80, lowLimit 5 and both
// limitEnable booleans true; it returns the folder's name and the extension's handle.
func (d *dRig) alarmedAmps() (string, string) {
	d.T.Helper()
	d.Station.SetSupertype("alarm:LimitEnable", "baja:Struct")
	d.afterAdd(func(st *boxstation.Station, n *boxstation.Node) {
		if n.Type == "alarm:OutOfRangeAlarmExt" && n.Child("offnormalAlgorithm") == nil {
			st.Component(n, "offnormalAlgorithm", "alarm:OffnormalAlgorithm", "")
		}
	})
	d.Station.Do(func(st *boxstation.Station) {
		st.AfterSet = func(_ *boxstation.Station, n *boxstation.Node) { // a swapped-in algorithm has its struct
			if n.Type == "alarm:OutOfRangeAlgorithm" && n.Child("limitEnable") == nil {
				n.Children = append(n.Children, &boxstation.Node{Name: "limitEnable", Type: "alarm:LimitEnable",
					Children: []*boxstation.Node{{Name: "lowLimitEnable", Type: "baja:Boolean", Value: "false"},
						{Name: "highLimitEnable", Type: "baja:Boolean", Value: "false"}}})
			}
		}
	})
	amps := d.Add("3", "Amps", "baja:Folder")
	alarm := d.Add(amps.Handle, "Alarm", "alarm:OutOfRangeAlarmExt")
	d.Set(alarm.Handle, "offnormalAlgorithm", map[string]any{"nm": "p", "t": "alarm:OutOfRangeAlgorithm", "v": ""})
	d.Set(alarm.Handle, "offnormalAlgorithm/highLimit", double("80.0"))
	d.Set(alarm.Handle, "offnormalAlgorithm/lowLimit", double("5.0"))
	for _, b := range []string{"highLimitEnable", "lowLimitEnable"} {
		d.Set(alarm.Handle, "offnormalAlgorithm/limitEnable/"+b, map[string]any{"nm": "p", "t": "baja:Boolean", "v": "true"})
	}
	return amps.Name, alarm.Handle
}

func TestARollbackKeepsAStructInsideItsParentAndRestoresItsBooleans(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(structContracts())
	nn, _ := d.alarmedAmps()
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	dry := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	for _, c := range objects(planOf(dry)["components"]) {
		if c["n"] == "limitEnable" {
			t.Errorf("limitEnable is planned as a component: %v", c)
		}
	}
	back := d.rollback(removed["batch_id"])
	if lost, ok := back["frozen_config_not_restored"]; ok {
		t.Errorf("frozen_config_not_restored = %v", lost)
	}
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	alarm := d.node(nn, "Alarm")
	for slot, want := range map[string]string{"highLimitEnable": "true", "lowLimitEnable": "true"} {
		if got := d.value(alarm.Handle, "offnormalAlgorithm", "limitEnable", slot); got != want {
			t.Errorf("restored %s = %v, want %s", slot, got, want)
		}
	}
	for slot, want := range map[string]string{"highLimit": "80.0", "lowLimit": "5.0"} {
		if got := d.value(alarm.Handle, "offnormalAlgorithm", slot); got != want {
			t.Errorf("restored %s = %v, want %s", slot, got, want)
		}
	}
}

// The struct's booleans are inside the snapshot (SnapshotDepth counts component levels), so the
// remove plan does not list them as cut, and a compact dry run counts three components, not four.
func TestAStructIsPartOfItsParentsBodyInTheRemoveSnapshot(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContractsOf(structContracts())
	nn, _ := d.alarmedAmps()
	args := removeArgs(nn)
	args["compact"] = true
	dry := d.Srv.Dry("n4_remove_component", args)
	if cut, ok := planOf(dry)["snapshot_truncated"]; ok {
		t.Errorf("snapshot_truncated = %v", cut)
	}
	summary, _ := planOf(dry)["inverse_summary"].(map[string]any)
	if summary["components"] != 3.0 {
		t.Errorf("inverse_summary = %v, want 3 components", summary)
	}
}

// A station registry that does not describe the types leaves them to the naming rule, and the plan
// says which ones it decided.
func TestARollbackPlanNamesTheTypesTheNamingRuleClassified(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	dry := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	if got := noteWith(planOf(dry)["notes"], "naming rule", "kitControl:NumericConst"); len(got) != 1 {
		t.Errorf("notes = %v", planOf(dry)["notes"])
	}
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "verified" || d.node(nn, "Src") == nil {
		t.Errorf("back = %v", back)
	}
}

// A struct kept at its parent's depth can sit on the last loaded level: its own children were never
// read, so the remove plan reports it in snapshot_truncated instead of showing a complete snapshot;
// the limitEnable booleans on that level (simples) are captured and not reported.
func TestAStructAtTheLoadCutIsReportedAsTruncated(t *testing.T) {
	d := destructiveRig(t, nil)
	contracts := structContracts()
	contracts["x:Inner"] = map[string]any{"p": "baja:Struct"}
	d.withContractsOf(contracts)
	d.Station.SetSupertype("x:Inner", "baja:Struct")
	nn, alarm := d.alarmedAmps()
	d.Set(alarm, "offnormalAlgorithm/limitEnable/inner", map[string]any{"nm": "p", "t": "x:Inner", "v": ""})
	d.Set(alarm, "offnormalAlgorithm/limitEnable/inner/deepleaf", map[string]any{"nm": "p", "t": "baja:String", "v": "deep"})
	dry := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	cut, _ := planOf(dry)["snapshot_truncated"].(map[string]any)
	if !same(cut["paths"], []any{"Alarm/offnormalAlgorithm/limitEnable/inner"}) {
		t.Errorf("snapshot_truncated = %v, want only the inner struct (its children were not loaded)", cut)
	}
	if len(noteWith(planOf(dry)["notes"], "NOT captured")) != 1 {
		t.Errorf("notes = %v", planOf(dry)["notes"])
	}
}
