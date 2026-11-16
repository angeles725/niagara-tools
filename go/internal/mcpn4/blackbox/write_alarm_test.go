package blackbox

// n4_add_alarm end to end against the fake station: one batch adds the extension, swaps its
// algorithms by type and sets the limits by path; the read-back reads every slot; the rollback
// removes the extension (bench retro 2026-11-15 row 10).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// stationAlarmClasses is the class table of the fake station.
const stationAlarmClasses = "Slot Path\nslot:/Services/AlarmService/defaultAlarmClass\nslot:/Services/AlarmService/Critical\n"

// newAlarmRig is a history rig (writes allowed, BQL, point supertypes) whose station answers the
// alarm class query and, like a station, gives an out-of-range algorithm set by type its frozen
// limitEnable struct (BOutOfRangeAlgorithm.limitEnable, alarm:LimitEnable).
func newAlarmRig(t *testing.T) *historyRig {
	t.Helper()
	h := newHistoryRig(t, noHistoryIDs)
	routeOrds(h.sv, boxstation.OrdRoute{Needle: "alarm:AlarmClass", Body: []byte(stationAlarmClasses)})
	h.Station.Do(func(st *boxstation.Station) {
		st.AfterSet = func(st *boxstation.Station, n *boxstation.Node) {
			if strings.HasPrefix(n.Type, "alarm:OutOfRange") {
				n.Children = append(n.Children, &boxstation.Node{Name: "limitEnable", Type: "alarm:LimitEnable",
					Children: []*boxstation.Node{{Name: "lowLimitEnable", Type: "baja:Boolean", Value: "false"},
						{Name: "highLimitEnable", Type: "baja:Boolean", Value: "false"}}})
			}
		}
	})
	return h
}

// alarmNode is the node at the slot path under the component with handle h, nil when absent.
func alarmNode(r *Rig, h string, path ...string) (out *boxstation.Node) {
	r.Station.Do(func(st *boxstation.Station) {
		out = st.ByHandle(h)
		for _, p := range path {
			if out = out.Child(p); out == nil {
				return
			}
		}
	})
	return out
}

func TestAddAlarmCreatesAConfiguredOutOfRangeExtensionAndRollsBack(t *testing.T) {
	h := newAlarmRig(t)
	temp := h.Add("3", "Temp", "control:NumericWritable")
	out := h.Srv.RunWrite("n4_add_alarm", Args{"points": []any{Folder + "/Temp"}, "high_limit": 80, "deadband": 1,
		"high_limit_text": "%alarmData.sourceName% high: %alarmData.presentValue% > %alarmData.highLimit%",
		"to_normal_text":  "%alarmData.sourceName% back to normal", "alarm_class": "Critical",
		"alarm_enable": []any{"offnormal", "normal"}, "time_delay_ms": 60000})
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	for _, c := range []struct {
		path       []string
		typ, value string
	}{
		{[]string{"OutOfRangeAlarmExt"}, "alarm:AlarmSourceExt", ""},
		{[]string{"OutOfRangeAlarmExt", "offnormalAlgorithm"}, "alarm:OutOfRangeAlgorithm", ""},
		{[]string{"OutOfRangeAlarmExt", "faultAlgorithm"}, "alarm:OutOfRangeFaultAlgorithm", ""},
		{[]string{"OutOfRangeAlarmExt", "offnormalAlgorithm", "highLimit"}, "baja:Double", "80.0"},
		{[]string{"OutOfRangeAlarmExt", "offnormalAlgorithm", "limitEnable", "highLimitEnable"}, "baja:Boolean", "true"},
		{[]string{"OutOfRangeAlarmExt", "offnormalAlgorithm", "limitEnable", "lowLimitEnable"}, "baja:Boolean", "false"},
		{[]string{"OutOfRangeAlarmExt", "alarmEnable"}, "alarm:AlarmTransitionBits", "5"},
		{[]string{"OutOfRangeAlarmExt", "alarmClass"}, "baja:String", "Critical"},
		{[]string{"OutOfRangeAlarmExt", "timeDelay"}, "baja:RelTime", "60000"},
	} {
		n := alarmNode(h.Rig, temp.Handle, c.path...)
		if n == nil || n.Type != c.typ || (c.value != "" && n.Value != c.value) {
			t.Errorf("%s = %+v, want %s %q", strings.Join(c.path, "/"), n, c.typ, c.value)
		}
	}
	plan := h.Srv.Dry("n4_rollback", Args{"batch_id": out["batch_id"]})
	h.Srv.OK("n4_rollback", Execute(Args{"batch_id": out["batch_id"]}, plan["confirmation_token"]))
	if hasChild(h.Rig, temp.Handle, "OutOfRangeAlarmExt") {
		t.Error("the rollback left the extension")
	}
}

func TestAddAlarmRefusesAtTheDryRunAndWritesNothing(t *testing.T) {
	h := newAlarmRig(t)
	fan := h.Add("3", "Fan", "control:BooleanWritable")
	for _, c := range []struct {
		args Args
		want string
	}{
		{Args{"kind": "out_of_range"}, "a boolean point (control:BooleanWritable): use one of boolean_change_of_state, " +
			"boolean_command_failure, status, or leave kind out"},
		{Args{"alarm_class": "Nope"}, "alarm class 'Nope' does not exist on the station; existing classes: " +
			"defaultAlarmClass, Critical."},
	} {
		c.args["points"] = []any{Folder + "/Fan"}
		if text := h.Srv.Err("n4_add_alarm", c.args); !strings.Contains(text, c.want) {
			t.Errorf("refusal = %q, want %q", text, c.want)
		}
	}
	h.Srv.RunWrite("n4_add_alarm", Args{"points": []any{Folder + "/Fan"}, "alarm_value": false})
	text := h.Srv.Err("n4_add_alarm", Args{"points": []any{Folder + "/Fan"}, "ext_name": "Second"})
	if !strings.Contains(text, "already has a boolean_change_of_state alarm extension, BooleanChangeOfStateAlarmExt") {
		t.Errorf("second: %q", text)
	}
	if out := h.Srv.RunWrite("n4_add_alarm", Args{"points": []any{Folder + "/Fan"}, "ext_name": "Second",
		"allow_second": true}); out["verdict"] != "verified" {
		t.Errorf("allow_second: %v", out)
	}
	if n := alarmNode(h.Rig, fan.Handle, "BooleanChangeOfStateAlarmExt", "offnormalAlgorithm", "alarmValue"); n == nil || n.Value != "false" {
		t.Errorf("alarmValue = %+v", n)
	}
}

func TestAddAlarmSetsAFaultLimitOnTheFaultAlgorithm(t *testing.T) {
	h := newAlarmRig(t)
	temp := h.Add("3", "Temp", "control:NumericWritable")
	out := h.Srv.RunWrite("n4_add_alarm", Args{"points": []any{Folder + "/Temp"}, "fault_high_limit": 100,
		"to_fault_text": "%alarmData.sourceName% sensor fault"})
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	for path, want := range map[string]string{"highLimit": "100.0", "limitEnable/highLimitEnable": "true",
		"limitEnable/lowLimitEnable": "false"} {
		n := alarmNode(h.Rig, temp.Handle, append([]string{"OutOfRangeAlarmExt", "faultAlgorithm"}, strings.Split(path, "/")...)...)
		if n == nil || n.Value != want {
			t.Errorf("faultAlgorithm/%s = %+v, want %q", path, n, want)
		}
	}
}
