package blackbox

// n4_station_health drivers=true: driver health per network, from the rows the tool already reads.

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Two networks: Bacnet holds a down device and a down+stale proxy point; Modbus holds a disabled
// device and a fault proxy point. The control points with control:NullProxyExt (one under a
// Modbus device, one under /Logic) are no driver points and count nowhere.
const (
	drvNetworksCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/BacnetNetwork,BacnetNetwork,bacnet:BacnetNetwork,0,true\n" +
		"slot:/Drivers/ModbusNetwork,ModbusNetwork,modbusAsync:ModbusAsyncNetwork,0,true\n"
	drvDevicesCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/BacnetNetwork/localDevice,localDevice,bacnet:LocalBacnetDevice,4,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1,Ahu1,bacnet:BacnetDevice,4,true\n" +
		"slot:/Drivers/ModbusNetwork/Meter1,Meter1,modbusAsync:ModbusAsyncDevice,0,true\n" +
		"slot:/Drivers/ModbusNetwork/Meter2,Meter2,modbusAsync:ModbusAsyncDevice,1,false\n"
	drvPointsCSV = "Slot Path,Name,Type,Bits,Type\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1/points/Temp,Temp,control:NumericPoint,20,bacnet:BacnetNumericProxyExt\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1/points/Fan,Fan,control:BooleanPoint,0,bacnet:BacnetBooleanProxyExt\n" +
		"slot:/Drivers/ModbusNetwork/Meter1/points/kW,kW,control:NumericPoint,2,modbusAsync:ModbusClientNumericProxyExt\n" +
		"slot:/Drivers/ModbusNetwork/Meter1/points/kWh,kWh,control:NumericPoint,0,modbusAsync:ModbusClientNumericProxyExt\n" +
		"slot:/Drivers/ModbusNetwork/Meter1/points/Calc,Calc,control:NumericWritable,2,control:NullProxyExt\n" +
		"slot:/Logic/Setpoint,Setpoint,control:NumericWritable,2,control:NullProxyExt\n"
)

func drvRig(t *testing.T) *Rig {
	t.Helper()
	r, sv := shxHealthRig(t)
	shxRouteHealth(sv, drvNetworksCSV, drvDevicesCSV, drvPointsCSV)
	return r
}

func drvDevices(total, down, fault, disabled float64) map[string]any {
	return map[string]any{"total": total, "down": down, "fault": fault, "disabled": disabled, "unknown": 0.0}
}

func drvPoints(total, stale, fault, down float64) map[string]any {
	return map[string]any{"total": total, "stale": stale, "fault": fault, "down": down, "disabled": 0.0, "unknown": 0.0}
}

func TestStationHealthDriversCountsDevicesAndProxyPointsPerNetwork(t *testing.T) {
	out := r0(drvRig(t), Args{"drivers": true})
	drivers := shxSection(t, out, "drivers")
	want := []any{
		map[string]any{"network": "slot:/Drivers/BacnetNetwork", "name": "BacnetNetwork", "type": "bacnet:BacnetNetwork",
			"status": []any{}, "enabled": true, "devices": drvDevices(1, 1, 0, 0),
			"devices_bad":           []any{map[string]any{"device": "slot:/Drivers/BacnetNetwork/Ahu1", "status": []any{"down"}}},
			"devices_bad_truncated": false, "points": drvPoints(2, 1, 0, 1),
			"points_bad":           []any{map[string]any{"point": "slot:/Drivers/BacnetNetwork/Ahu1/points/Temp", "status": []any{"down", "stale"}}},
			"points_bad_truncated": false},
		map[string]any{"network": "slot:/Drivers/ModbusNetwork", "name": "ModbusNetwork", "type": "modbusAsync:ModbusAsyncNetwork",
			"status": []any{}, "enabled": true, "devices": map[string]any{"total": 2.0, "down": 0.0, "fault": 0.0, "disabled": 1.0, "unknown": 0.0},
			"devices_bad": []any{map[string]any{"device": "slot:/Drivers/ModbusNetwork/Meter2", "status": []any{"disabled"},
				"enabled": false}},
			"devices_bad_truncated": false, "points": drvPoints(2, 0, 1, 0),
			"points_bad":           []any{map[string]any{"point": "slot:/Drivers/ModbusNetwork/Meter1/points/kW", "status": []any{"fault"}}},
			"points_bad_truncated": false},
	}
	if got := drivers["networks"]; !reflect.DeepEqual(got, want) {
		t.Errorf("networks =\n%v\nwant\n%v", got, want)
	}
	wantTotals := map[string]any{"networks": 2.0, "devices": map[string]any{"total": 3.0, "down": 1.0, "fault": 0.0,
		"disabled": 1.0, "unknown": 0.0}, "points": drvPoints(4, 1, 1, 1), "unassigned_points": 0.0,
		"proxy_unknown": 0.0, "rows_truncated": false}
	if got := drivers["totals"]; !reflect.DeepEqual(got, wantTotals) {
		t.Errorf("totals = %v", got)
	}
}

// Without drivers (or with drivers=false) the reply is the one the tool gave before it.
func TestStationHealthWithoutDriversKeepsItsReply(t *testing.T) {
	want := []string{"capacity", "capacity_truncated", "devices", "elapsed_ms", "license", "networks", "points", "system", "totals"}
	for _, args := range []Args{nil, {"drivers": false}} {
		out := r0(drvRig(t), args)
		keys := make([]string, 0, len(out))
		for k := range out {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(keys, want) {
			t.Errorf("args %v: keys = %v", args, keys)
		}
	}
}

func TestStationHealthDriversCapsTheBadListsAndSaysSo(t *testing.T) {
	drivers := shxSection(t, r0(drvRig(t), Args{"drivers": true, "max_bad": 0}), "drivers")
	bacnet := objects(drivers["networks"])[0]
	if len(objects(bacnet["points_bad"])) != 0 || bacnet["points_bad_truncated"] != true ||
		len(objects(bacnet["devices_bad"])) != 0 || bacnet["devices_bad_truncated"] != true ||
		!reflect.DeepEqual(bacnet["points"], drvPoints(2, 1, 0, 1)) {
		t.Errorf("bacnet = %v", bacnet)
	}
}

func TestStationHealthDriversIsNullWithANoteWhenASectionFailed(t *testing.T) {
	r, sv := shxHealthRig(t)
	shxRouteHealth(sv, drvNetworksCSV, "", drvPointsCSV)
	out := r0(r, Args{"drivers": true})
	note, _ := out["drivers_note"].(string)
	if drivers, present := out["drivers"]; !present || drivers != nil || !strings.Contains(note, "devices") {
		t.Errorf("drivers %v, note %q", out["drivers"], note)
	}
}

// A proxy point under no network and a point whose proxy type the station did not answer are
// counted apart, never dropped silently; an unreadable status is unknown, never ok.
func TestStationHealthDriversCountsWhatItCannotPlace(t *testing.T) {
	r, sv := shxHealthRig(t)
	shxRouteHealth(sv, drvNetworksCSV, drvDevicesCSV, "Slot Path,Name,Type,Bits,Type\n"+
		"slot:/Elsewhere/P,P,control:NumericPoint,0,bacnet:BacnetNumericProxyExt\n"+
		"slot:/Drivers/BacnetNetwork/Ahu1/points/Q,Q,control:NumericPoint,0,\n"+
		"slot:/Drivers/BacnetNetwork/Ahu1/points/R,R,control:NumericPoint,,bacnet:BacnetNumericProxyExt\n")
	drivers := shxSection(t, r0(r, Args{"drivers": true}), "drivers")
	totals := drivers["totals"].(map[string]any)
	if totals["unassigned_points"] != 1.0 || totals["proxy_unknown"] != 1.0 ||
		totals["points"].(map[string]any)["unknown"] != 1.0 || totals["points"].(map[string]any)["total"] != 1.0 {
		t.Errorf("totals = %v", totals)
	}
	bad := objects(objects(drivers["networks"])[0]["points_bad"])
	if len(bad) != 1 || !reflect.DeepEqual(bad[0]["status"], []any{"unknown"}) {
		t.Errorf("points_bad = %v", bad)
	}
}

func r0(r *Rig, args Args) map[string]any { return r.Srv.OK("n4_station_health", args) }
