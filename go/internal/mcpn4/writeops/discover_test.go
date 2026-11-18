package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const discJob = "station:|slot:/Services/JobService/BacnetDiscoverDevicesJob3"

// clock is a write state whose sleeps advance a fake monotonic clock.
func clock() *plan.State {
	now := 0.0
	return &plan.State{SaveInterval: 1, Sleep: func(s float64) { now += s }, Monotonic: func() float64 { return now }}
}

func discNetwork() *pyjson.Object {
	return obj(bacnetNet, obj("", obj("h", "3a", "t", "bacnet:BacnetNetwork")))
}

func discPlan(t *testing.T, args *pyjson.Object) *plan.Planned {
	t.Helper()
	p, err := writeops.DiscoverDevices.PlanWith(&loads{nodes: discNetwork()}, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// The default discover is the Workbench "Send Global": all networks, every instance, 20 s (a 10 s wait missed MS/TP Spyders, retro 2026-11-16 P2).
func TestDiscoverDefaultsToAGlobalWhoIsOverEveryInstance(t *testing.T) {
	p := discPlan(t, obj("network_ord", bacnetNet))
	want := `{"ssc": "invokeAction", "arg": {"h": "3a", "a": "submitDeviceManagerJob", "b": {"nm": "p", "t": "bacnet:DeviceDiscoveryConfig", "s": [` +
		`{"nm": "p", "n": "deviceLowLimit", "t": "baja:Integer", "v": "0"}, ` +
		`{"nm": "p", "n": "deviceHighLimit", "t": "baja:Integer", "v": "4194302"}, ` +
		`{"nm": "p", "n": "networks", "t": "bacnet:DiscoveryNetworks", "v": "all;"}, ` +
		`{"nm": "p", "n": "waitResponseTime", "t": "baja:Integer", "v": "20"}]}}}`
	if got := js(t, p.Ops[0]); got != want {
		t.Errorf("op =\n%s\nwant\n%s", got, want)
	}
	notes := js(t, p.Notes)
	for _, part := range []string{"global", "no device is added", "address", "no inverse"} {
		if !strings.Contains(notes, part) {
			t.Errorf("notes miss %q: %s", part, notes)
		}
	}
	if len(p.Inverse) != 0 {
		t.Errorf("inverse = %v", p.Inverse)
	}
}

// Listed networks and an instance range are sent as the station encodes them.
func TestDiscoverListedNetworksAndRange(t *testing.T) {
	p := discPlan(t, obj("network_ord", bacnetNet, "networks", []any{2, 5}, "low_instance", 2000,
		"high_instance", 2099, "wait_seconds", 20))
	got := js(t, p.Ops[0])
	for _, part := range []string{`"v": "-;2,5"`, `"v": "2000"`, `"v": "2099"`, `"v": "20"`} {
		if !strings.Contains(got, part) {
			t.Errorf("op misses %s: %s", part, got)
		}
	}
}

func TestDiscoverRefusals(t *testing.T) {
	for _, tc := range []struct {
		args *pyjson.Object
		want string
	}{
		{obj("network_ord", bacnetNet, "low_instance", 10, "high_instance", 5), "low_instance"},
		{obj("network_ord", bacnetNet, "high_instance", 4194303), "high_instance"},
		{obj("network_ord", bacnetNet, "wait_seconds", 0), "wait_seconds"},
		{obj("network_ord", bacnetNet, "wait_seconds", 61), "wait_seconds"},
		{obj("network_ord", bacnetNet, "networks", []any{0}), "networks"},
		{obj("network_ord", bacnetNet, "networks", []any{65535}), "networks"},
		{obj("network_ord", bacnetNet, "networks", "2"), "networks"},
	} {
		_, err := writeops.DiscoverDevices.PlanWith(&loads{nodes: discNetwork()}, tc.args, clock(), nil)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: err = %v, want one naming %s", js(t, tc.args), err, tc.want)
		}
	}
	modbus := "station:|slot:/Drivers/ModbusAsyncNetwork"
	c := &loads{nodes: obj(modbus, obj("", obj("h", "4", "t", "modbusAsync:ModbusAsyncNetwork")))}
	if _, err := writeops.DiscoverDevices.PlanWith(c, obj("network_ord", modbus), clock(), nil); err == nil ||
		!strings.Contains(err.Error(), "bacnet:BacnetNetwork") {
		t.Errorf("Modbus network: err = %v", err)
	}
}

// discoverStation is a network with two registered devices (one at the address it answers from,
// one silent) and a finished job with three rows: a registered device, a new one with no name and
// a registered instance answering from another address.
func discoverStation(state string) *loads {
	row := func(n, id, net, mac, name string) []any {
		kv := []any{n, obj("t", "bacnet:DiscoveryDevice", "d", "name="+name+" id="+id+" addr="+net+":"+mac),
			n + "/objectId", obj("t", "bacnet:BacnetObjectIdentifier", "v", id),
			n + "/vendorId", obj("t", "baja:Integer", "v", "17"),
			n + "/address", obj("t", "bacnet:BacnetAddress", "d", net+":"+mac),
			n + "/address/networkNumber", obj("t", "baja:Integer", "v", net),
			n + "/address/macAddress", obj("t", "bacnet:BacnetOctetString", "v", mac),
			n + "/vendorName", obj("t", "baja:String", "v", "Honeywell"),
			n + "/modelName", obj("t", "baja:String", "v", "Spyder")}
		if name != "" {
			kv = append(kv, n+"/deviceName", obj("t", "baja:String", "v", name))
		}
		return kv
	}
	job := []any{"", obj("h", "9", "t", "bacnet:BacnetDiscoverDevicesJob"),
		"jobState", obj("t", "baja:JobState", "v", state)}
	job = append(job, row("DiscoveryDevice", "device:2004", "2", "04", "$32004")...)
	job = append(job, "DiscoveryDevice/listSize", obj("t", "baja:Integer", "v", "46"),
		"DiscoveryDevice/address/addressType", obj("t", "baja:Integer", "v", "3"))
	job = append(job, row("DiscoveryDevice1", "device:2012", "2", "0c", "")...)
	job = append(job, row("DiscoveryDevice2", "device:2009", "2", "1f", "$32009")...)
	dev := func(name, mac, id string) []any {
		return []any{name, obj("h", "d"+name, "t", "honeywellBacnetSpyder:BacnetSpyder"),
			name + "/address", obj("t", "bacnet:BacnetAddress", "d", "2:"+mac)}
	}
	net := []any{"", obj("h", "3a", "t", "bacnet:BacnetNetwork"), "localDevice", obj("h", "5", "t", "bacnet:LocalBacnetDevice")}
	net = append(net, dev("$32004", "04", "device:2004")...)
	net = append(net, dev("$32009", "09", "device:2009")...)
	net = append(net, dev("$32010", "0a", "device:2010")...)
	objectID := func(id string) *pyjson.Object {
		return obj("", obj("t", "bacnet:BacnetDeviceObject"), "objectId", obj("t", "bacnet:BacnetObjectIdentifier", "v", id))
	}
	return &loads{nodes: obj(discJob, obj(job...), bacnetNet, obj(net...),
		bacnetNet+"/$32004/config/deviceObject", objectID("device:2004"),
		bacnetNet+"/$32009/config/deviceObject", objectID("device:2009"),
		bacnetNet+"/$32010/config/deviceObject", objectID("device:2010"))}
}

func discReadback(t *testing.T, c *loads, reply any) (*pyjson.Object, string) {
	t.Helper()
	args := obj("network_ord", bacnetNet)
	p, err := writeops.DiscoverDevices.PlanWith(&loads{nodes: discNetwork()}, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, observed, verdict, err := writeops.DiscoverDevices.Readback(c, args, p, []any{reply}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return observed.(*pyjson.Object), verdict
}

func TestDiscoverReadbackRowsAgainstTheDatabase(t *testing.T) {
	ord := obj("t", "baja:Ord", "v", "slot:/Services/JobService/BacnetDiscoverDevicesJob3")
	o, verdict := discReadback(t, discoverStation("success"), ord)
	if verdict != "verified" {
		t.Errorf("verdict = %s", verdict)
	}
	got := js(t, o)
	for _, part := range []string{
		`"job": "` + discJob + `"`, `"job_state": "success"`, `"found": 3`,
		`{"instance": 2004, "network": 2, "mac": "04", "mac_display": "4", "name": "2004", "vendor_id": 17, "vendor": "Honeywell", "model": "Spyder", "objects": 46, "in_database": "$32004", "status": "registered"}`,
		`{"instance": 2012, "network": 2, "mac": "0c", "mac_display": "12", "name": "", "vendor_id": 17, "vendor": "Honeywell", "model": "Spyder", "objects": null, "in_database": null, "status": "new"}`,
		`"status": "address_differs", "database_address": "2:09"`,
		`"not_answering": [{"name": "$32010", "instance": 2010, "address": "2:0a"}]`,
	} {
		if !strings.Contains(got, part) {
			t.Errorf("observed misses %s:\n%s", part, got)
		}
	}
}

// The reply may also be the plain ORD text; a job that never finishes is unverified, a failed one
// a mismatch, and a reply without an ORD unverified with a hint.
func TestDiscoverReadbackJobStates(t *testing.T) {
	if _, v := discReadback(t, discoverStation("success"), "slot:/Services/JobService/BacnetDiscoverDevicesJob3"); v != "verified" {
		t.Errorf("plain ORD reply: verdict = %s", v)
	}
	if o, v := discReadback(t, discoverStation("running"), discJob); v != "unverified" || !strings.Contains(js(t, o), "still running") {
		t.Errorf("running: %s %s", v, js(t, o))
	}
	if _, v := discReadback(t, discoverStation("failed"), discJob); v != "mismatch" {
		t.Errorf("failed: verdict = %s", v)
	}
	if o, v := discReadback(t, discoverStation("success"), nil); v != "unverified" || !strings.Contains(js(t, o), "no job ORD") {
		t.Errorf("null reply: %s %s", v, js(t, o))
	}
}

// The MAC is also given as Workbench's Device Manager shows it: an MS/TP station number in
// decimal ("0f" is 15), a BACnet/IP address as a.b.c.d:port, anything else as the station's octets.
func TestDiscoverMacDisplayLikeWorkbench(t *testing.T) {
	for _, tc := range []struct{ kind, mac, want string }{
		{"3", "0f", "15"}, {"", "0f", "15"}, {"2", "c0 a8 01 0a ba c0", "192.168.1.10:47808"},
		{"", "c0 a8 01 0a ba c0", "192.168.1.10:47808"}, {"1", "00 11 22 33 44 55", "00 11 22 33 44 55"},
	} {
		if got := writeops.MacDisplay(tc.kind, tc.mac); got != tc.want {
			t.Errorf("type %q mac %q: %q, want %q", tc.kind, tc.mac, got, tc.want)
		}
	}
}
