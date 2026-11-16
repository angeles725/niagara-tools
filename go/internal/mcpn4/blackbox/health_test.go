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
	want := map[string]any{"total": 5.0, "ok": 2.0, "null_only": 0.0, "proxy": 4.0, "by_flag": map[string]any{"down": 1.0, "fault": 1.0, "stale": 2.0},
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
	wantNets := map[string]any{"total": 2.0, "ok": 1.0, "null_only": 0.0, "disabled": 1.0, "by_flag": map[string]any{"disabled": 1.0},
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
		"unassigned_points": 1.0, "rows_truncated": false}
	if got := shxSection(t, out, "totals"); !reflect.DeepEqual(got, wantTotals) {
		t.Errorf("totals = %v", got)
	}
}

func TestStationHealthReadsCapacityFromTheSpyMetricsPage(t *testing.T) {
	r, sv := shxHealthRig(t)
	capacity := shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")
	resources := objects(capacity["resources"])
	if len(resources) != 6 || !reflect.DeepEqual(resources[2], map[string]any{"type": "Points", "used": 36.0,
		"limit": nil, "unlimited": true, "live": 4.0}) {
		t.Errorf("resources = %v", capacity["resources"])
	}
	if !reflect.DeepEqual(capacity["recount"], map[string]any{"recountLastRun": "never", "recountLastFail": "never",
		"recountLastFailReason": ""}) || capacity["source"] != "spy:/metrics" {
		t.Errorf("capacity = %v", capacity)
	}
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics-licensed.html")})
	resources = objects(shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")["resources"])
	if !reflect.DeepEqual(resources[2], map[string]any{"type": "Points", "used": 1036.0, "limit": 1250.0, "unlimited": false, "live": 4.0}) {
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
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, "Slot Path,Name,Type,Bits,Type\nslot:/A,A,control:NumericPoint,,x:Ext\n"+
		"slot:/B,B,control:NumericPoint,0,x:Ext\n")
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

// A status whose only flag is null is a point with no value (an unlinked input, for example):
// counted under null_only, never listed as not ok. null beside another flag is not ok.
func TestStationHealthCountsANullOnlyStatusApartFromTheNotOkOnes(t *testing.T) {
	r, sv := shxHealthRig(t)
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, "Slot Path,Name,Type,Bits,Type\n"+
		"slot:/Logic/Ok,Ok,control:NumericWritable,0,control:NullProxyExt\n"+
		"slot:/Logic/Unlinked,Unlinked,control:NumericWritable,64,control:NullProxyExt\n"+
		"slot:/Logic/NullStale,NullStale,control:NumericPoint,80,x:Ext\n"+
		"slot:/Logic/Unread,Unread,control:NumericPoint,,x:Ext\n")
	points := shxSection(t, r.Srv.OK("n4_station_health", nil), "points")
	want := map[string]any{"total": 4.0, "ok": 1.0, "null_only": 1.0, "proxy": 2.0, "not_ok_count": 2.0,
		"by_flag": map[string]any{"null": 2.0, "stale": 1.0, "unknown": 1.0}, "truncated": false, "rows_truncated": false,
		"not_ok": []any{map[string]any{"path": "slot:/Logic/NullStale", "flags": []any{"stale", "null"}},
			map[string]any{"path": "slot:/Logic/Unread", "flags": []any{"unknown"}}}}
	if !reflect.DeepEqual(points, want) {
		t.Errorf("points = %v", points)
	}
	if sum := points["ok"].(float64) + points["null_only"].(float64) + points["not_ok_count"].(float64); sum != points["total"] {
		t.Errorf("ok + null_only + not_ok_count = %v, total %v", sum, points["total"])
	}
	for _, name := range []string{"networks", "devices"} {
		s := shxSection(t, r.Srv.OK("n4_station_health", nil), name)
		if s["null_only"] != 0.0 || s["ok"].(float64)+s["not_ok_count"].(float64) != s["total"] {
			t.Errorf("%s = %v", name, s)
		}
	}
}

// Item 6: the capacity is never silently partial.
func TestStationHealthCapacityIsNeverSilentlyPartial(t *testing.T) {
	r, sv := shxHealthRig(t)
	six := ""
	for _, kind := range []string{"Networks", "Devices", "Points", "Links", "Histories", "Schedules"} {
		six += shxCapacityRow(kind, "none", "7")
	}
	filler := func(rows int) string { return "<table>" + strings.Repeat("<tr><td>x</td></tr>", rows) + "</table>" }
	for _, ahead := range []int{499, 600, 5000} { // rows ahead of the table never hide a resource
		shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage(filler(ahead), "never", six)})
		out := r.Srv.OK("n4_station_health", nil)
		if got := objects(shxSection(t, out, "capacity")["resources"]); len(got) != 6 || out["capacity_truncated"] != false {
			t.Errorf("%d rows ahead: %d resources, capacity_truncated %v, note %v", ahead, len(got), out["capacity_truncated"], out["capacity_note"])
		}
	}
	// a page cut before the table: unknown and said so, never "has no table"
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage(filler(120000), "never", six)})
	out := r.Srv.OK("n4_station_health", nil)
	note, _ := out["capacity_note"].(string)
	if out["capacity"] != nil || out["capacity_truncated"] != true || !strings.Contains(note, "cut") ||
		strings.Contains(note, "no Global Capacity table") {
		t.Errorf("cut before the table: capacity %v, truncated %v, note %q", out["capacity"], out["capacity_truncated"], note)
	}
	// a page cut inside the table: what was read, marked partial
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage("", "never", six+"<tr><td>Tail</td><td>none</td><td>"+strings.Repeat("9", 3<<20)+"</td></tr>")})
	out = r.Srv.OK("n4_station_health", nil)
	note, _ = out["capacity_note"].(string)
	if got := objects(shxSection(t, out, "capacity")["resources"]); len(got) < 6 || out["capacity_truncated"] != true || !strings.Contains(note, "max_bytes") {
		t.Errorf("cut inside the table: %d resources, truncated %v, note %q", len(got), out["capacity_truncated"], note)
	}
	// a row that does not match the header is counted in the note, not dropped silently
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage("", "never", six+"<tr><td>Odd</td><td>1</td></tr><tr><td>Odd2</td></tr>")})
	out = r.Srv.OK("n4_station_health", nil)
	note, _ = out["capacity_note"].(string)
	if got := objects(shxSection(t, out, "capacity")["resources"]); len(got) != 6 || !strings.Contains(note, "2 row(s)") || out["capacity_truncated"] != false {
		t.Errorf("ragged rows: %d resources, note %q", len(got), note)
	}
}

// Item 7: totals say when a query that feeds them was cut.
func TestStationHealthTotalsSayWhenAQueryFeedingThemWasCut(t *testing.T) {
	r, _ := shxHealthRig(t)
	if got := shxSection(t, r.Srv.OK("n4_station_health", nil), "totals")["rows_truncated"]; got != false {
		t.Errorf("whole queries: totals.rows_truncated = %v", got)
	}
	if got := shxSection(t, r.Srv.OK("n4_station_health", Args{"max_rows": 3}), "totals")["rows_truncated"]; got != true {
		t.Errorf("a cut query: totals.rows_truncated = %v", got)
	}
}

// Item 8a: proxy counts the points whose proxy extension is not control:NullProxyExt.
func TestStationHealthCountsTheProxyPointsApartFromEveryControlPoint(t *testing.T) {
	r, sv := shxHealthRig(t)
	points := shxSection(t, r.Srv.OK("n4_station_health", nil), "points")
	if points["total"] != 5.0 || points["proxy"] != 4.0 {
		t.Errorf("points = %v", points)
	}
	if _, has := points["proxy_unknown"]; has {
		t.Errorf("proxy_unknown without an unanswered cell: %v", points)
	}
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, "Slot Path,Name,Type,Bits,Type\nslot:/A,A,control:NumericPoint,0,\n"+
		"slot:/B,B,control:NumericPoint,0,x:Ext\nslot:/C,C,control:NumericPoint,0,control:NullProxyExt\n")
	points = shxSection(t, r.Srv.OK("n4_station_health", nil), "points")
	if points["proxy"] != 1.0 || points["proxy_unknown"] != 1.0 {
		t.Errorf("an unanswered proxy type: %v", points)
	}
}

// Item 8b and 8c: the licence tally beside the live count, and the warning that it only grows.
func TestStationHealthPutsTheLiveCountBesideTheLicenceTally(t *testing.T) {
	r, sv := shxHealthRig(t)
	rows := shxCapacityRow("Networks", "none", "3") + shxCapacityRow("Devices", "50", "4") +
		shxCapacityRow("Points", "1,250", "4") + shxCapacityRow("Links", "none", "94")
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage("", "15-Nov-26 7:05 PM", rows)})
	capacity := shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")
	want := []any{
		map[string]any{"type": "Networks", "used": 3.0, "limit": nil, "unlimited": true, "live": 2.0},
		map[string]any{"type": "Devices", "used": 4.0, "limit": 50.0, "unlimited": false, "live": 4.0},
		map[string]any{"type": "Points", "used": 4.0, "limit": 1250.0, "unlimited": false, "live": 4.0},
		map[string]any{"type": "Links", "used": 94.0, "limit": nil, "unlimited": true, "live": nil}}
	note, _ := capacity["inflation_note"].(string)
	if !reflect.DeepEqual(capacity["resources"], want) || capacity["may_be_inflated"] != true || !strings.Contains(note, "recount") {
		t.Errorf("used above live: %v", capacity)
	}
	// no used above its live count and a recount that ran: not flagged
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage("", "15-Nov-26 7:05 PM",
		shxCapacityRow("Networks", "none", "2")+shxCapacityRow("Points", "none", "4"))})
	capacity = shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity")
	if _, has := capacity["inflation_note"]; has || capacity["may_be_inflated"] != false {
		t.Errorf("level counts after a recount: %v", capacity)
	}
	// the same counts with a recount that never ran: flagged
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxMetricsPage("", "never", shxCapacityRow("Networks", "none", "2"))})
	if capacity = shxSection(t, r.Srv.OK("n4_station_health", nil), "capacity"); capacity["may_be_inflated"] != true {
		t.Errorf("recount never ran: %v", capacity)
	}
	// a subtree's counts are not the station's: no live count beside the tally
	capacity = shxSection(t, r.Srv.OK("n4_station_health", Args{"base": "station:|slot:/Drivers"}), "capacity")
	if got := objects(capacity["resources"])[0]; got["live"] != nil {
		t.Errorf("a base below the root: %v", got)
	}
}
