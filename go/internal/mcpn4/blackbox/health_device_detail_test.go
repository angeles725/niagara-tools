package blackbox

// n4_station_health drivers=true device_detail=true: per bad device what a technician needs to
// fix it, and per network a bus or single-device pattern hint (field retro delta U12).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// A serial Modbus network with 3 of its 4 devices down, and a BACnet network with 1 of its 3
// down. Meter3 is missing from the detail answer, so its detail fields are unreadable.
const (
	ddNetworksCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/Rs485,Rs485,modbusAsync:ModbusAsyncNetwork,0,true\n" +
		"slot:/Drivers/BacnetNetwork,BacnetNetwork,bacnet:BacnetNetwork,0,true\n"
	ddDevicesCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/Rs485/Meter1,Meter1,modbusAsync:ModbusAsyncDevice,4,true\n" +
		"slot:/Drivers/Rs485/Meter2,Meter2,modbusAsync:ModbusAsyncDevice,4,true\n" +
		"slot:/Drivers/Rs485/Meter3,Meter3,modbusAsync:ModbusAsyncDevice,4,true\n" +
		"slot:/Drivers/Rs485/Meter4,Meter4,modbusAsync:ModbusAsyncDevice,0,true\n" +
		"slot:/Drivers/BacnetNetwork/localDevice,localDevice,bacnet:LocalBacnetDevice,0,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1,Ahu1,bacnet:BacnetDevice,0,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2,Ahu2,bacnet:BacnetDevice,4,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu3,Ahu3,bacnet:BacnetDevice,0,true\n"
	ddPointsCSV = "Slot Path,Name,Type,Bits,Type\n" +
		"slot:/Drivers/Rs485/Meter1/points/kW,kW,control:NumericPoint,20,modbusAsync:ModbusClientNumericProxyExt\n" +
		"slot:/Drivers/Rs485/Meter1/points/kWh,kWh,control:NumericPoint,16,modbusAsync:ModbusClientNumericProxyExt\n" +
		"slot:/Drivers/Rs485/Meter1/points/Calc,Calc,control:NumericWritable,16,control:NullProxyExt\n" +
		"slot:/Drivers/Rs485/Meter2/points/kW,kW,control:NumericPoint,4,modbusAsync:ModbusClientNumericProxyExt\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2/points/Temp,Temp,control:NumericPoint,20,bacnet:BacnetNumericProxyExt\n"
	ddDetailCSV = "Slot Path,Type,Fault Cause,Last Ok Time,Last Fail Time,Last Fail Cause,Device Address,Address\n" +
		"slot:/Drivers/Rs485/Meter1,modbusAsync:ModbusAsyncDevice,,12-Nov-26 3:04 PM CST,17-Nov-26 9:00 AM CST,Response timeout,11,\n" +
		"slot:/Drivers/Rs485/Meter2,modbusAsync:ModbusAsyncDevice,Config fault,null,17-Nov-26 9:01 AM CST,Response timeout,12,\n" +
		"slot:/Drivers/Rs485/Meter4,modbusAsync:ModbusAsyncDevice,,17-Nov-26 9:05 AM CST,null,,14,\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2,bacnet:BacnetDevice,,16-Nov-26 8:00 PM CST,17-Nov-26 9:02 AM CST,Stack:timeout,,1:0a\n"
	ddPortsCSV = "Slot Path,Port Name\n" +
		"slot:/Drivers/Rs485,COM2\n" +
		"slot:/Drivers/BacnetNetwork,\n"
)

// ddRig answers the three health queries and the two detail queries (the detail needles first:
// the health needles are in them too).
func ddRig(t *testing.T, detail, ports string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := shxHealthRig(t)
	routeOrds(sv, boxstation.OrdRoute{Needle: "health.lastFailCause", Body: []byte(detail)},
		boxstation.OrdRoute{Needle: "serialPortConfig", Body: []byte(ports)},
		boxstation.OrdRoute{Needle: "driver:DeviceNetwork", Body: []byte(ddNetworksCSV)},
		boxstation.OrdRoute{Needle: "driver:Device", Body: []byte(ddDevicesCSV)},
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte(ddPointsCSV)})
	return r, sv
}

func ddNetworks(t *testing.T, out map[string]any) []map[string]any {
	t.Helper()
	return objects(shxSection(t, out, "drivers")["networks"])
}

func ddBad(device, fault, okTime, failTime, cause, address any, stale float64) map[string]any {
	return map[string]any{"device": device, "status": []any{"down"}, "fault_cause": fault, "last_fail_cause": cause,
		"last_ok": okTime, "last_fail": failTime, "address": address, "stale_points": stale}
}

func TestStationHealthDeviceDetailNamesTheBusWhenAllButOneDeviceIsDown(t *testing.T) {
	r, _ := ddRig(t, ddDetailCSV, ddPortsCSV)
	rs485 := ddNetworks(t, r0(r, Args{"drivers": true, "device_detail": true}))[0]
	want := []any{
		ddBad("slot:/Drivers/Rs485/Meter1", nil, "12-Nov-26 3:04 PM CST", "17-Nov-26 9:00 AM CST", "Response timeout", "11", 2),
		ddBad("slot:/Drivers/Rs485/Meter2", "Config fault", nil, "17-Nov-26 9:01 AM CST", "Response timeout", "12", 0),
		ddBad("slot:/Drivers/Rs485/Meter3", nil, nil, nil, nil, nil, 0),
	}
	if got := rs485["devices_bad"]; !reflect.DeepEqual(got, want) {
		t.Errorf("devices_bad =\n%v\nwant\n%v", got, want)
	}
	wantPattern := map[string]any{"kind": "bus", "down": 3.0, "total": 4.0,
		"hint": "check bus wiring, termination, polarity and the port"}
	if !reflect.DeepEqual(rs485["pattern"], wantPattern) || rs485["port"] != "COM2" {
		t.Errorf("pattern %v, port %v", rs485["pattern"], rs485["port"])
	}
}

func TestStationHealthDeviceDetailNamesTheSingleDownDevice(t *testing.T) {
	r, _ := ddRig(t, ddDetailCSV, ddPortsCSV)
	bacnet := ddNetworks(t, r0(r, Args{"drivers": true, "device_detail": true}))[1]
	want := []any{ddBad("slot:/Drivers/BacnetNetwork/Ahu2", nil, "16-Nov-26 8:00 PM CST", "17-Nov-26 9:02 AM CST",
		"Stack:timeout", "1:0a", 1)}
	wantPattern := map[string]any{"kind": "single", "down": 1.0, "total": 3.0, "hint": "check that device's address and power"}
	if got := bacnet["devices_bad"]; !reflect.DeepEqual(got, want) {
		t.Errorf("devices_bad = %v", got)
	}
	if !reflect.DeepEqual(bacnet["pattern"], wantPattern) || bacnet["port"] != nil {
		t.Errorf("pattern %v, port %v", bacnet["pattern"], bacnet["port"])
	}
}

// A failed detail query leaves every detail field null and says so, never a guessed value; the
// counts and the stale points (from rows already read) still answer.
func TestStationHealthDeviceDetailIsNullWhenTheStationDoesNotAnswerIt(t *testing.T) {
	r, _ := ddRig(t, "", "")
	out := r0(r, Args{"drivers": true, "device_detail": true})
	drivers := shxSection(t, out, "drivers")
	meter1 := objects(objects(drivers["networks"])[0]["devices_bad"])[0]
	if meter1["fault_cause"] != nil || meter1["last_fail_cause"] != nil || meter1["last_ok"] != nil ||
		meter1["address"] != nil || meter1["stale_points"] != 2.0 {
		t.Errorf("meter1 = %v", meter1)
	}
	note, _ := drivers["device_detail_note"].(string)
	if !strings.Contains(note, "device detail query failed") || !strings.Contains(note, "port query failed") {
		t.Errorf("device_detail_note = %q", note)
	}
}

func TestStationHealthRefusesDeviceDetailWithoutDrivers(t *testing.T) {
	r, sv := ddRig(t, ddDetailCSV, ddPortsCSV)
	for _, args := range []Args{{"device_detail": true}, {"drivers": false, "device_detail": true}} {
		if text := r.Srv.Err("n4_station_health", args); !strings.Contains(text, "device_detail=true needs drivers=true") {
			t.Errorf("args %v: %q", args, text)
		}
	}
	if n := len(ordRequests(sv)); n != 0 {
		t.Errorf("a refused call sent %d ORD request(s)", n)
	}
}

// drivers=true without device_detail (or with it false) is the reply and the requests of before.
func TestStationHealthDriversWithoutDeviceDetailIsUnchanged(t *testing.T) {
	base, baseSv := ddRig(t, ddDetailCSV, ddPortsCSV)
	want := r0(base, Args{"drivers": true})["drivers"]
	wantRequests := ordRequests(baseSv)
	for _, req := range wantRequests {
		if strings.Contains(req, "health.lastFailCause") || strings.Contains(req, "serialPortConfig") {
			t.Fatalf("drivers=true alone sent a detail query: %s", req)
		}
	}
	r, sv := ddRig(t, ddDetailCSV, ddPortsCSV)
	if got := r0(r, Args{"drivers": true, "device_detail": false})["drivers"]; !reflect.DeepEqual(got, want) {
		t.Errorf("drivers =\n%v\nwant\n%v", got, want)
	}
	if got := ordRequests(sv); len(got) != len(wantRequests) {
		t.Errorf("requests = %d, want %d", len(got), len(wantRequests))
	}
	for _, n := range objects(want.(map[string]any)["networks"]) {
		if _, has := n["pattern"]; has {
			t.Errorf("pattern without device_detail: %v", n)
		}
	}
}

// An answer without the address columns still places the BDevice slots; address is null.
func TestStationHealthDeviceDetailWithoutAddressColumnsKeepsTheOthers(t *testing.T) {
	r, _ := ddRig(t, "Slot Path,Type,Fault Cause,Last Ok Time,Last Fail Time,Last Fail Cause\n"+
		"slot:/Drivers/BacnetNetwork/Ahu2,bacnet:BacnetDevice,,null,17-Nov-26 9:02 AM CST,Stack:timeout\n", ddPortsCSV)
	drivers := shxSection(t, r0(r, Args{"drivers": true, "device_detail": true}), "drivers")
	ahu2 := objects(objects(drivers["networks"])[1]["devices_bad"])[0]
	if ahu2["last_fail_cause"] != "Stack:timeout" || ahu2["last_ok"] != nil || ahu2["address"] != nil {
		t.Errorf("ahu2 = %v", ahu2)
	}
	if note, _ := drivers["device_detail_note"].(string); !strings.Contains(note, "no address columns") {
		t.Errorf("device_detail_note = %q", note)
	}
}
