package blackbox

// n4_commissioning_report, the points and histories sections: every finding is a row the
// station answered.

import (
	"strings"
	"testing"
)

const comDev1 = "slot:/Drivers/NetA/Dev1/points/"

var comStatusTables = map[string][]string{
	"networks": {"slot:/Drivers/NetA,NetA,bacnet:BacnetNetwork,0,true", "slot:/Drivers/NetB,NetB,modbusAsync:ModbusAsyncNetwork,2,true"},
	"devices": {"slot:/Drivers/NetA/Dev1,Dev1,bacnet:BacnetDevice,0,true", "slot:/Drivers/NetA/Dev2,Dev2,bacnet:BacnetDevice,4,true",
		"slot:/Drivers/NetA/Dev3,Dev3,bacnet:BacnetDevice,1,false"},
	"points": {comDev1 + "Ok,Ok,Ok,control:NumericPoint,0", comDev1 + "Flt,Flt,Fan fault,control:BooleanWritable,2",
		comDev1 + "Dwn,Dwn,Dwn,control:NumericPoint,4", comDev1 + "Stl,Stl,Stl,control:NumericPoint,16",
		comDev1 + "Dis,Dis,Dis,control:NumericPoint,1", comDev1 + "Ovr,Ovr,Ovr,control:NumericWritable,32",
		comDev1 + "Alm,Alm,Alm,control:NumericPoint,8", comDev1 + "Nul,Nul,Nul,control:NumericWritable,64",
		"slot:/Drivers/NetA/Dev2/points/A,A,A,control:NumericPoint,20", "slot:/Drivers/NetA/Dev2/points/B,B,B,control:NumericPoint,20",
		"slot:/Drivers/NetA/Dev3/points/C,C,C,control:NumericPoint,1", "slot:/Logic/Bad,Bad,Bad,control:NumericPoint,what"},
}

func TestCommissioningCountsPointsPerStatusAndListsTheOffenders(t *testing.T) {
	r, _ := comRig(t, comStatusTables)
	s := comSection(t, r, "points", nil)
	fault := comWant(t, s, "in_fault", 1, comDev1+"Flt")
	if item := objects(fault["items"])[0]; item["name"] != "Fan fault" || item["fact"] != "status fault" {
		t.Errorf("fault item = %v", item)
	}
	comWant(t, s, "overridden", 1, comDev1+"Ovr")
	comWant(t, s, "in_alarm", 1, comDev1+"Alm")
	comWant(t, s, "null_output", 1, comDev1+"Nul")
	comWant(t, s, "status_unreadable", 1, "slot:/Logic/Bad")
	if s["state"] != "findings" || s["total_points"] != 12.0 || s["findings"].(float64) < 10 {
		t.Errorf("section = %v", s)
	}
}

func TestCommissioningListsDownOrDisabledDevicesFirstAndCountsTheirPoints(t *testing.T) {
	r, _ := comRig(t, comStatusTables)
	s := comSection(t, r, "points", nil)
	devices := comWant(t, s, "devices_down_or_disabled", 2, "slot:/Drivers/NetA/Dev2", "slot:/Drivers/NetA/Dev3")
	if fact := comFact(devices, "slot:/Drivers/NetA/Dev2"); fact != "status down; 2 point(s) below it, 2 not ok" {
		t.Errorf("Dev2 fact = %q", fact)
	}
	if fact := comFact(devices, "slot:/Drivers/NetA/Dev3"); fact != "status disabled, enabled false; 1 point(s) below it, 1 not ok" {
		t.Errorf("Dev3 fact = %q", fact)
	}
	// the points of a listed device are counted in their status, never listed
	down := comWant(t, s, "down", 3, comDev1+"Dwn")
	stale := comWant(t, s, "stale", 3, comDev1+"Stl")
	disabled := comWant(t, s, "disabled", 2, comDev1+"Dis")
	if down["under_listed_devices"] != 2.0 || stale["under_listed_devices"] != 2.0 || disabled["under_listed_devices"] != 1.0 || down["more"] != 0.0 {
		t.Errorf("down %v stale %v disabled %v", down, stale, disabled)
	}
	comWant(t, s, "networks_not_ok", 1, "slot:/Drivers/NetB")
	nets := objects(s["by_network"])
	if len(nets) != 2 || nets[0]["path"] != "slot:/Drivers/NetA" || nets[0]["devices"] != 3.0 || nets[0]["devices_listed"] != 2.0 ||
		nets[0]["points"] != 11.0 || nets[0]["points_not_ok"] != 10.0 || nets[1]["points"] != 0.0 {
		t.Errorf("by_network = %v", s["by_network"])
	}
	text := r.Srv.Call("n4_commissioning_report", Args{"sections": []any{"points"}}).Text
	if first := strings.Index(text, `"devices_down_or_disabled"`); first < 0 || first > strings.Index(text, `"in_fault"`) {
		t.Errorf("devices are not first: %s", text)
	}
}

var comHistoryTables = map[string][]string{
	"points": {"slot:/Logic/Temp,Temp,Temp,control:NumericPoint,0", "slot:/Logic/Fan,Fan,Fan,control:BooleanWritable,0",
		"slot:/Logic/Mode,Mode,Mode,control:EnumWritable,0", "slot:/Logic/Zone$20Sp,Zone Sp,Zone Sp,control:NumericWritable,0"},
	"histories": {"slot:/Logic/Temp/NumericInterval,NumericInterval,history:NumericIntervalHistoryExt,true,0,/Lab/Temp,15-Nov-26 7:05 PM CST",
		"slot:/Logic/Zone$20Sp/Cov,Cov,history:NumericCovHistoryExt,false,2,/Lab/ZoneSp,null"},
}

func TestCommissioningListsThePointsWithoutAHistoryExtension(t *testing.T) {
	r, _ := comRig(t, comHistoryTables)
	s := comSection(t, r, "histories", nil)
	// without expect_histories the absence is a count, not a list and not a finding
	c := comWant(t, s, "points_without_history", 2)
	if c["note"] != "pass expect_histories to list them" || c["more"] != 0.0 || s["findings"] != 3.0 {
		t.Errorf("without the filter: %v, findings %v", c, s["findings"])
	}
	s = comSection(t, r, "histories", Args{"expect_histories": Args{}})
	c = comWant(t, s, "points_without_history", 2, "slot:/Logic/Fan", "slot:/Logic/Mode")
	if fact := comFact(c, "slot:/Logic/Fan"); fact != "no history extension below it" || c["note"] != nil || s["findings"] != 5.0 {
		t.Errorf("fact = %q, check %v, findings %v", fact, c, s["findings"])
	}
	s = comSection(t, r, "histories", Args{"expect_histories": Args{"kinds": []any{"boolean"}}})
	comWant(t, s, "points_without_history", 1, "slot:/Logic/Fan")
	s = comSection(t, r, "histories", Args{"expect_histories": Args{"name_pattern": "M*"}})
	comWant(t, s, "points_without_history", 1, "slot:/Logic/Mode")
	for _, bad := range []Args{{"kinds": []any{"analog"}}, {"name_pattern": "[x"}, {"pattern": "x"}} {
		if msg := r.Srv.Err("n4_commissioning_report", Args{"expect_histories": bad}); !strings.Contains(msg, "expect_histories") {
			t.Errorf("%v: %s", bad, msg)
		}
	}
}

func TestCommissioningListsHistoryExtensionsDisabledInFaultOrNeverRecorded(t *testing.T) {
	r, sv := comRig(t, comHistoryTables)
	s := comSection(t, r, "histories", nil)
	ext := "slot:/Logic/Zone$20Sp/Cov"
	if fact := comFact(comWant(t, s, "disabled", 1, ext), ext); fact != "enabled false (history id /Lab/ZoneSp)" {
		t.Errorf("disabled fact = %q", fact)
	}
	comWant(t, s, "in_fault", 1, ext)
	if fact := comFact(comWant(t, s, "never_recorded", 1, ext), ext); fact != "lastRecord.timestamp is null (history id /Lab/ZoneSp)" {
		t.Errorf("never recorded fact = %q", fact)
	}
	// a station that answers no lastRecord cell proves nothing about records
	comRoutes(sv, map[string][]string{"histories": {"slot:/Logic/Temp/H,H,history:NumericIntervalHistoryExt,true,0,/Lab/Temp,"}})
	s = comSection(t, r, "histories", nil)
	if why, _ := comCheck(t, s, "never_recorded")["not_checked"].(string); !strings.Contains(why, "lastRecord") || s["state"] != "partial" {
		t.Errorf("never_recorded = %v, state %v", comCheck(t, s, "never_recorded"), s["state"])
	}
}
