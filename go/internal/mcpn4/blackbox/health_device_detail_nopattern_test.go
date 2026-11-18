package blackbox

// A network with no pattern (every device ok) next to one with down devices: its pattern is a
// JSON null, never a typed nil that crashes the encoder (live lab finding on #813).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestStationHealthDeviceDetailNetworkWithoutPatternIsNull(t *testing.T) {
	r, sv := shxHealthRig(t)
	networks := ddNetworksCSV + "slot:/Drivers/Lon,Lon,lonworks:LonNetwork,0,true\n"
	devices := ddDevicesCSV + "slot:/Drivers/Lon/Vav1,Vav1,lonworks:LonDevice,0,true\n" +
		"slot:/Drivers/Lon/Vav2,Vav2,lonworks:LonDevice,0,true\n"
	routeOrds(sv, boxstation.OrdRoute{Needle: "health.lastFailCause", Body: []byte(ddDetailCSV)},
		boxstation.OrdRoute{Needle: "serialPortConfig", Body: []byte(ddPortsCSV)},
		boxstation.OrdRoute{Needle: "driver:DeviceNetwork", Body: []byte(networks)},
		boxstation.OrdRoute{Needle: "driver:Device", Body: []byte(devices)},
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte(ddPointsCSV)})
	nets := ddNetworks(t, r0(r, Args{"drivers": true, "device_detail": true}))
	if len(nets) != 3 {
		t.Fatalf("networks = %v", nets)
	}
	lon := nets[2]
	if p, has := lon["pattern"]; !has || p != nil {
		t.Errorf("Lon pattern = %v (present %v), want null", p, has)
	}
}

// A network without a serial port: BQL answers NULL for the missing path, which is a null port,
// and a mostly-down IP network gets the network-path hint, not the wiring one (live lab finding).
func TestStationHealthDeviceDetailIPNetworkHasNullPortAndNetworkHint(t *testing.T) {
	ports := "Slot Path,Port Name\nslot:/Drivers/Rs485,COM2\nslot:/Drivers/BacnetNetwork,NULL\n"
	devices := strings.NewReplacer("Ahu1,bacnet:BacnetDevice,0", "Ahu1,bacnet:BacnetDevice,4",
		"Ahu3,bacnet:BacnetDevice,0", "Ahu3,bacnet:BacnetDevice,4").Replace(ddDevicesCSV) // 3 of 3 down
	r, sv := shxHealthRig(t)
	routeOrds(sv, boxstation.OrdRoute{Needle: "health.lastFailCause", Body: []byte(ddDetailCSV)},
		boxstation.OrdRoute{Needle: "serialPortConfig", Body: []byte(ports)},
		boxstation.OrdRoute{Needle: "driver:DeviceNetwork", Body: []byte(ddNetworksCSV)},
		boxstation.OrdRoute{Needle: "driver:Device", Body: []byte(devices)},
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte(ddPointsCSV)})
	bacnet := ddNetworks(t, r0(r, Args{"drivers": true, "device_detail": true}))[1]
	if p, has := bacnet["port"]; !has || p != nil {
		t.Errorf("BacnetNetwork port = %v, want null", p)
	}
	pattern, _ := bacnet["pattern"].(map[string]any)
	if pattern["kind"] != "bus" || !strings.Contains(pattern["hint"].(string), "network path") {
		t.Errorf("BacnetNetwork pattern = %v", bacnet["pattern"])
	}
}
