package bql_test

// Go twin of test_bql.TestPureHelpers.test_local_devices_are_flagged_and_counted_apart
// (go_server.go_twin("bql", "unit")).

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestLocalDevicesAreFlaggedAndCountedApart(t *testing.T) {
	dev := func(name string) *pyjson.Object {
		o := pyjson.NewObject()
		o.Set("Slot Path", "slot:/Drivers/SnmpNetwork/"+name)
		return o
	}
	if !bql.IsLocalDevice(dev("localDevice")) || bql.IsLocalDevice(dev("Ups1")) {
		t.Error("IsLocalDevice")
	}
	_, devices, _ := parse(t, fixture(t, "bql_devices.csv"), 50, nil)
	_, points, _ := parse(t, fixture(t, "bql_points.csv"), 50, nil)
	_, nets, _ := parse(t, fixture(t, "bql_networks.csv"), 50, nil)
	inv, err := bql.SummarizeInventory("/Drivers", nets, devices, points)
	if err != nil {
		t.Fatal(err)
	}
	if got := pyjson.Repr(get(inv, "totals")); got != "{'networks': 4, 'field_devices': 4, 'local_devices': 1, 'points': 5, 'unassigned_points': 0}" {
		t.Errorf("totals %s", got)
	}
	find := func(key, list, name string) *pyjson.Object {
		for _, e := range get(inv, list).([]any) {
			if get(e.(*pyjson.Object), key) == name {
				return e.(*pyjson.Object)
			}
		}
		t.Fatalf("no %s %s", list, name)
		return nil
	}
	snmp := find("network", "networks", "SnmpNetwork")
	if get(snmp, "field_devices") != 1 || get(snmp, "local_devices") != 1 || get(snmp, "points") != 1 {
		t.Errorf("SnmpNetwork %s", pyjson.Repr(snmp))
	}
	if local := find("name", "devices", "localDevice"); get(local, "local") != true {
		t.Errorf("localDevice %s", pyjson.Repr(local))
	}
	if meter := find("name", "devices", "Meter A"); get(meter, "network") != "ModbusTcpNetwork" || get(meter, "points") != 2 {
		t.Errorf("Meter A %s", pyjson.Repr(meter))
	}
	if empty := find("network", "networks", "NiagaraNetwork"); get(empty, "field_devices") != 0 || get(empty, "points") != 0 {
		t.Errorf("NiagaraNetwork %s", pyjson.Repr(empty))
	}
}
