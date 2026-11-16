package blackbox

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The answers of the three health queries, as ITableToCsv writes them: the status bits are the
// decimal of baja:Status (0 ok, 2 fault, 4 down, 16 stale, 1 disabled); a point row ends with the
// type of its proxy extension (the second Type column), control:NullProxyExt for a point that
// is no proxy.
const (
	shxNetworksCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/BacnetNetwork,BacnetNetwork,bacnet:BacnetNetwork,0,true\n" +
		"slot:/Drivers/ModbusNetwork,ModbusNetwork,modbusAsync:ModbusAsyncNetwork,1,false\n"
	shxDevicesCSV = "Slot Path,Name,Type,Bits,Enabled\n" +
		"slot:/Drivers/BacnetNetwork/localDevice,localDevice,bacnet:LocalBacnetDevice,0,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1,Ahu1,bacnet:BacnetDevice,0,true\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2,Ahu2,bacnet:BacnetDevice,4,true\n" +
		"slot:/Drivers/ModbusNetwork/Meter,Meter,modbusAsync:ModbusAsyncDevice,5,true\n"
	shxPointsCSV = "Slot Path,Name,Type,Bits,Type\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1/points/Temp,Temp,control:NumericPoint,0,bacnet:BacnetNumericProxyExt\n" +
		"slot:/Drivers/BacnetNetwork/Ahu1/points/Fan,Fan,control:BooleanWritable,0,bacnet:BacnetBooleanProxyExt\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2/points/Temp,Temp,control:NumericPoint,20,bacnet:BacnetNumericProxyExt\n" +
		"slot:/Drivers/BacnetNetwork/Ahu2/points/Fan,Fan,control:BooleanWritable,16,bacnet:BacnetBooleanProxyExt\n" +
		"slot:/Logic/Setpoint,Setpoint,control:NumericWritable,2,control:NullProxyExt\n"
)

// shxHealthRig is a connected servlet rig whose station answers everything n4_station_health
// reads: the SystemService slots, the three status queries and the recorded spy metrics page.
func shxHealthRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	shxSystemService(r.Station, map[string][2]string{
		"stationName": {"baja:String", "LabStation"}, "hostId": {"baja:String", "Lab-0000-0000"},
		"model": {"baja:String", "Workstation"}, "niagaraVersion": {"baja:String", "4.15.6.20"},
		"osName": {"baja:String", "Linux"}, "javaVmVersion": {"baja:String", "25.0"},
		"systemTime": {"baja:AbsTime", "2026-11-15T19:05:43.877-06:00"}, "numCpus": {"baja:Integer", "16"},
		"currentCpuUsage": {"baja:Integer", "85"}, "overallCpuUsage": {"baja:Integer", "65"},
		"totalPhysicalMemory": {"baja:Integer", "8000"}, "freePhysicalMemory": {"baja:Integer", "5000"},
		"bajaHome": {"baja:String", "/opt/niagara"}})
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, shxPointsCSV)
	shxServeSpy(sv, "metrics", boxstation.SpyPage{Body: shxSpyFixture(t, "metrics.html")})
	return r, sv
}

// shxSystemService gives the station a Services/PlatformServices/SystemService with slots
// (name: type, value).
func shxSystemService(st *boxstation.Station, slots map[string][2]string) {
	st.Do(func(st *boxstation.Station) {
		node := st.Root()
		for _, name := range []string{"Services", "PlatformServices", "SystemService"} {
			node = st.Component(node, name, "baja:Component", "")
		}
		for name, tv := range slots {
			node.Children = append(node.Children, &boxstation.Node{Name: name, Type: tv[0], Value: tv[1]})
		}
	})
}

// shxRouteHealth answers the network, device and point queries (the network needle first: the
// device needle is a prefix of it).
func shxRouteHealth(sv *boxstation.Servlets, networks, devices, points string) {
	routeOrds(sv, boxstation.OrdRoute{Needle: "driver:DeviceNetwork", Body: []byte(networks)},
		boxstation.OrdRoute{Needle: "driver:Device", Body: []byte(devices)},
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte(points)})
}

func shxSection(t *testing.T, out map[string]any, name string) map[string]any {
	t.Helper()
	m, ok := out[name].(map[string]any)
	if !ok {
		t.Fatalf("%s = %v", name, out[name])
	}
	return m
}

// shxMetricsPage is a spy metrics page in the measured layout: lead (markup placed ahead of the
// tables), the Recount table with lastRun, then the Global Capacity table with rows as HTML.
func shxMetricsPage(lead, lastRun, rows string) []byte {
	return []byte("<html><body>" + lead + "<table><tr><th colspan='3'>Recount</th></tr><tr><td>recountLastRun</td><td>" +
		lastRun + "</td></tr></table><table><tr><th colspan='3'>Global Capacity</th></tr>" +
		"<tr><th>Type</th><th>Limit</th><th>Used</th></tr>" + rows + "</table></body></html>")
}

func shxCapacityRow(kind, limit, used string) string {
	return "<tr><td>" + kind + "</td><td>" + limit + "</td><td>" + used + "</td></tr>"
}
