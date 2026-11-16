package blackbox

// n4_station_health: the station's state in one call, each section on its own.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestStationHealthAnswersTheSystemValuesWithADerivedMemoryPercentage(t *testing.T) {
	r, _ := shxHealthRig(t)
	want := map[string]any{"stationName": "LabStation", "hostId": "Lab-0000-0000", "model": "Workstation",
		"niagaraVersion": "4.15.6.20", "osName": "Linux", "javaVmVersion": "25.0",
		"systemTime": "2026-11-15T19:05:43.877-06:00", "numCpus": 16.0, "currentCpuUsage": 85.0,
		"overallCpuUsage": 65.0, "totalPhysicalMemory": 8000.0, "freePhysicalMemory": 5000.0,
		"memory_used_percent": 37.5}
	out := r.Srv.OK("n4_station_health", nil)
	if got := shxSection(t, out, "system"); !reflect.DeepEqual(got, want) {
		t.Errorf("system = %v", got)
	}
	if ms, ok := out["elapsed_ms"].(float64); !ok || ms < 0 {
		t.Errorf("elapsed_ms = %v", out["elapsed_ms"])
	}
}

func TestStationHealthCountsPointsPerStatusFlagAndNamesTheNotOkOnes(t *testing.T) {
	r, _ := shxHealthRig(t)
	want := map[string]any{"total": 5.0, "ok": 2.0, "by_flag": map[string]any{"down": 1.0, "fault": 1.0, "stale": 2.0},
		"not_ok_count": 3.0, "truncated": false, "rows_truncated": false, "not_ok": []any{
			map[string]any{"path": "slot:/Drivers/BacnetNetwork/Ahu2/points/Temp", "flags": []any{"down", "stale"}},
			map[string]any{"path": "slot:/Drivers/BacnetNetwork/Ahu2/points/Fan", "flags": []any{"stale"}},
			map[string]any{"path": "slot:/Logic/Setpoint", "flags": []any{"fault"}}}}
	if got := shxSection(t, r.Srv.OK("n4_station_health", nil), "points"); !reflect.DeepEqual(got, want) {
		t.Errorf("points = %v", got)
	}
}

func TestStationHealthNamesTheNetworksAndDevicesThatAreNotOk(t *testing.T) {
	r, _ := shxHealthRig(t)
	out := r.Srv.OK("n4_station_health", nil)
	wantNets := map[string]any{"total": 2.0, "ok": 1.0, "disabled": 1.0, "by_flag": map[string]any{"disabled": 1.0},
		"not_ok_count": 1.0, "truncated": false, "rows_truncated": false, "not_ok": []any{map[string]any{
			"path": "slot:/Drivers/ModbusNetwork", "name": "ModbusNetwork", "type": "modbusAsync:ModbusAsyncNetwork",
			"flags": []any{"disabled"}, "enabled": false}}}
	if got := shxSection(t, out, "networks"); !reflect.DeepEqual(got, wantNets) {
		t.Errorf("networks = %v", got)
	}
	devices := shxSection(t, out, "devices")
	if devices["total"] != 4.0 || devices["ok"] != 2.0 || devices["disabled"] != 0.0 ||
		!reflect.DeepEqual(devices["by_flag"], map[string]any{"disabled": 1.0, "down": 2.0}) ||
		!reflect.DeepEqual(sortedStrings(devices["not_ok"], "name"), []string{"Ahu2", "Meter"}) {
		t.Errorf("devices = %v", devices)
	}
	wantTotals := map[string]any{"networks": 2.0, "field_devices": 3.0, "local_devices": 1.0, "points": 5.0,
		"unassigned_points": 1.0}
	if got := shxSection(t, out, "totals"); !reflect.DeepEqual(got, wantTotals) {
		t.Errorf("totals = %v", got)
	}
}

func TestStationHealthReadsCapacityFromTheSpyMetricsPage(t *testing.T) {
	r, sv := shxHealthRig(t)
	capacity := shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")
	resources := objects(capacity["resources"])
	if len(resources) != 6 || !reflect.DeepEqual(resources[2], map[string]any{"type": "Points", "used": 36.0,
		"limit": nil, "unlimited": true}) {
		t.Errorf("resources = %v", capacity["resources"])
	}
	if !reflect.DeepEqual(capacity["recount"], map[string]any{"recountLastRun": "never", "recountLastFail": "never",
		"recountLastFailReason": ""}) || capacity["source"] != "spy:/metrics" {
		t.Errorf("capacity = %v", capacity)
	}
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics-licensed.html")})
	resources = objects(shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")["resources"])
	if !reflect.DeepEqual(resources[2], map[string]any{"type": "Points", "used": 1036.0, "limit": 1250.0, "unlimited": false}) {
		t.Errorf("licensed points = %v", resources[2])
	}
}

func TestStationHealthCapacityIsNullWithANoteWhenTheSpyPageIsUnreachable(t *testing.T) {
	r, sv := shxHealthRig(t)
	for status, word := range map[int]string{403: "forbidden", 404: "not_found"} {
		shxServeSpy(sv, "metrics", boxstation.SpyPage{Status: status})
		out := r.Srv.OK("n4_station_health", nil)
		capacity, present := out["capacity"]
		note, _ := out["capacity_note"].(string)
		if !present || capacity != nil || !strings.Contains(note, word) {
			t.Errorf("HTTP %d: capacity %v (present %v), note %q", status, capacity, present, note)
		}
		if shxSection(t, out, "points")["total"] != 5.0 {
			t.Errorf("HTTP %d: the other sections did not answer: %v", status, out)
		}
	}
}

func TestStationHealthCapacityIsNullWhenThePageHasNoCapacityRows(t *testing.T) {
	r, sv := shxHealthRig(t)
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics-no-capacity.html")})
	out := r.Srv.OK("n4_station_health", nil)
	note, _ := out["capacity_note"].(string)
	if capacity, present := out["capacity"]; !present || capacity != nil || !strings.Contains(note, "no Global Capacity table") {
		t.Errorf("capacity %v, note %q", out["capacity"], note)
	}
}

func TestStationHealthReportsAFailingSectionAndStillAnswersTheOthers(t *testing.T) {
	r, sv := servletRig(t, nil) // no SystemService in this station
	shxRouteHealth(sv, shxNetworksCSV, "", shxPointsCSV)
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics.html")})
	out := r.Srv.OK("n4_station_health", nil)
	for _, name := range []string{"system", "devices"} {
		section := shxSection(t, out, name)
		if text, _ := section["error"].(string); text == "" || len(section) != 1 {
			t.Errorf("%s = %v, want only its error", name, section)
		}
	}
	if text := shxSection(t, out, "devices")["error"].(string); !strings.Contains(text, "no header row") {
		t.Errorf("devices error = %q", text)
	}
	if shxSection(t, out, "networks")["total"] != 2.0 || shxSection(t, out, "points")["total"] != 5.0 ||
		len(objects(shxSection(t, out, "capacity")["resources"])) != 6 {
		t.Errorf("the other sections did not answer: %v", out)
	}
	note, _ := out["totals_note"].(string)
	if totals, present := out["totals"]; !present || totals != nil || !strings.Contains(note, "devices") {
		t.Errorf("totals %v, note %q", out["totals"], note)
	}
}

func TestStationHealthCapsTheNotOkListAndSaysSo(t *testing.T) {
	r, _ := shxHealthRig(t)
	points := shxSection(t, r.Srv.OK("n4_station_health", Args{"max_not_ok": 1}), "points")
	if points["truncated"] != true || points["not_ok_count"] != 3.0 || len(objects(points["not_ok"])) != 1 ||
		points["total"] != 5.0 {
		t.Errorf("points = %v", points)
	}
	points = shxSection(t, r.Srv.OK("n4_station_health", Args{"max_not_ok": 0}), "points")
	if points["truncated"] != true || len(objects(points["not_ok"])) != 0 {
		t.Errorf("max_not_ok 0: %v", points)
	}
}

func TestStationHealthMarksCountsOfACutQueryAsLowerBounds(t *testing.T) {
	r, _ := shxHealthRig(t)
	out := r.Srv.OK("n4_station_health", Args{"max_rows": 2})
	points := shxSection(t, out, "points")
	if points["rows_truncated"] != true || points["total"] != 2.0 {
		t.Errorf("points = %v", points)
	}
	if shxSection(t, out, "networks")["rows_truncated"] != false {
		t.Errorf("networks = %v", out["networks"])
	}
}

func TestStationHealthNeverCallsAnUnreadableStatusOk(t *testing.T) {
	r, sv := shxHealthRig(t)
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, "Slot Path,Name,Type,Bits\nslot:/A,A,control:NumericPoint,\n"+
		"slot:/B,B,control:NumericPoint,0\n")
	points := shxSection(t, r.Srv.OK("n4_station_health", nil), "points")
	if points["ok"] != 1.0 || !reflect.DeepEqual(points["by_flag"], map[string]any{"unknown": 1.0}) {
		t.Errorf("points = %v", points)
	}
}

func TestStationHealthWithARefusedLoginIsAnAuthError(t *testing.T) {
	r, sv := shxHealthRig(t)
	sv.SetPassword("changed")
	if text := r.Srv.Err("n4_station_health", nil); !strings.Contains(text, "401") {
		t.Errorf("error = %q", text)
	}
}

func TestStationHealthIsAReadToolThatNeedsASession(t *testing.T) {
	r := NewRig(t, nil)
	if tool := toolsByName(t, r.Srv)["n4_station_health"]; Dig(tool, "annotations", "readOnlyHint") != true {
		t.Errorf("tools/list entry = %v", tool)
	}
	if text := r.Srv.Err("n4_station_health", nil); !strings.Contains(text, "n4_connect") {
		t.Errorf("without a session: %q", text)
	}
}
