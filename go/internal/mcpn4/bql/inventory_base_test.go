package bql_test

// n4_inventory under a base above the driver container (second bench retro, delta 6): a network
// is the row of the networks query that holds a device or a point, at any depth below base.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// The tree of the observation: a site folder holding a plant folder with the driver container
// (five networks of different types) and two plain folders of logic beside the plant.
const (
	rfxContainer = "slot:/Site/Plant/Drivers"
	rfxNetworks  = "Slot Path,Name,Type\n" +
		"{D}/BacnetNetwork,BacnetNetwork,bacnet:BacnetNetwork\n" +
		"{D}/ModbusTcpNetwork,ModbusTcpNetwork,modbusTcp:ModbusTcpNetwork\n" +
		"{D}/LonNetwork,LonNetwork,lonworks:LonNetwork\n" +
		"{D}/SnmpNetwork,SnmpNetwork,snmp:SnmpNetwork\n" +
		"{D}/Niagara$20Network,Niagara$20Network,niagaraDriver:NiagaraNetwork\n"
	rfxDevices = "Slot Path,Name,Type\n" +
		"{D}/BacnetNetwork/Ahu1,Ahu1,bacnet:BacnetDevice\n" +
		"{D}/ModbusTcpNetwork/Meter,Meter,modbusTcp:ModbusTcpDevice\n" +
		"{D}/SnmpNetwork/localDevice,localDevice,snmp:SnmpAgent\n" +
		"{D}/SnmpNetwork/Ups,Ups,snmp:SnmpDevice\n"
	rfxPoints = "Slot Path,Name,Type,Out\n" +
		"{D}/BacnetNetwork/Ahu1/points/Fan,Fan,control:BooleanPoint,true {ok}\n" +
		"{D}/ModbusTcpNetwork/Meter/points/kW,kW,control:NumericPoint,12.5 {ok}\n" +
		"{D}/ModbusTcpNetwork/Meter/points/kWh,kWh,control:NumericPoint,3.0 {ok}\n" +
		"slot:/Site/Logic/Setpoint,Setpoint,control:NumericWritable,21.0 {ok}\n" +
		"slot:/Site/Schedules/Enable,Enable,control:BooleanWritable,true {ok}\n"
)

// rfxInventory is n4_inventory of base on a station answering the tree above.
func rfxInventory(t *testing.T, base string) *pyjson.Object {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	at := func(csv string) string { return strings.ReplaceAll(csv, "{D}", rfxContainer) }
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "driver:DeviceNetwork", Body: at(rfxNetworks)},
		{Needle: "driver:Device", Body: at(rfxDevices)}, {Needle: "control:ControlPoint", Body: at(rfxPoints)}}
	return ok(t, s, "n4_inventory", "base", base)
}

const (
	rfxWantNetworks = "[{'network': '{P}BacnetNetwork', 'type': 'bacnet:BacnetNetwork', 'field_devices': 1, 'local_devices': 0, 'points': 1}, " +
		"{'network': '{P}ModbusTcpNetwork', 'type': 'modbusTcp:ModbusTcpNetwork', 'field_devices': 1, 'local_devices': 0, 'points': 2}, " +
		"{'network': '{P}LonNetwork', 'type': 'lonworks:LonNetwork', 'field_devices': 0, 'local_devices': 0, 'points': 0}, " +
		"{'network': '{P}SnmpNetwork', 'type': 'snmp:SnmpNetwork', 'field_devices': 1, 'local_devices': 1, 'points': 0}, " +
		"{'network': '{P}Niagara Network', 'type': 'niagaraDriver:NiagaraNetwork', 'field_devices': 0, 'local_devices': 0, 'points': 0}]"
	rfxWantTotals = "{'networks': 5, 'field_devices': 3, 'local_devices': 1, 'points': 5, 'unassigned_points': 2}"
)

// rfxCheck compares the networks (named with prefix), the totals and each device's network.
func rfxCheck(t *testing.T, inv *pyjson.Object, prefix string) {
	t.Helper()
	if got, want := repr(get(inv, "networks")), strings.ReplaceAll(rfxWantNetworks, "{P}", prefix); got != want {
		t.Errorf("networks\n got %s\nwant %s", got, want)
	}
	if got := repr(get(inv, "totals")); got != rfxWantTotals {
		t.Errorf("totals\n got %s\nwant %s", got, rfxWantTotals)
	}
	var devices []string
	for _, d := range get(inv, "devices").([]any) {
		o := d.(*pyjson.Object)
		devices = append(devices, repr([]any{get(o, "name"), get(o, "network"), get(o, "local"), get(o, "points")}))
	}
	want := strings.ReplaceAll("['Ahu1', '{P}BacnetNetwork', False, 1] ['Meter', '{P}ModbusTcpNetwork', False, 2] "+
		"['localDevice', '{P}SnmpNetwork', True, 0] ['Ups', '{P}SnmpNetwork', False, 0]", "{P}", prefix)
	if got := strings.Join(devices, " "); got != want {
		t.Errorf("devices\n got %s\nwant %s", got, want)
	}
}

func TestInventoryFindsEachNetworkByItsRowUnderABaseAboveTheDriverContainer(t *testing.T) {
	rfxCheck(t, rfxInventory(t, "station:|slot:/Site"), "Plant/Drivers/")
}

func TestInventoryFromTheStationRootNamesNetworksByTheirPath(t *testing.T) {
	rfxCheck(t, rfxInventory(t, "station:|slot:/"), "Site/Plant/Drivers/")
}

func TestInventoryAtTheDriverContainerKeepsNetworkNames(t *testing.T) {
	rfxCheck(t, rfxInventory(t, "station:|slot:/Site/Plant/Drivers"), "")
}

func TestInventoryOfOneNetworkAsBaseNamesThatNetwork(t *testing.T) {
	inv := rfxInventory(t, "/Site/Plant/Drivers/SnmpNetwork") // the fake answers every row: the others are outside base
	if got := repr(get(inv, "networks")); got != "[{'network': 'SnmpNetwork', 'type': 'snmp:SnmpNetwork', 'field_devices': 1, 'local_devices': 1, 'points': 0}]" {
		t.Errorf("networks %s", got)
	}
}
