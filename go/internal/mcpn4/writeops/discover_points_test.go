package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const (
	ptsDevice = bacnetNet + "/Vav1"
	ptsExt    = ptsDevice + "/points"
	ptsJob    = "station:|slot:/Services/JobService/BacnetDiscoverPointsJob2"
)

func ptsStation(state string) *loads {
	row := func(n, name, id, value, desc string) []any {
		kv := []any{n, obj("h", "r"+n, "t", "bacnet:DiscoveryPoint"),
			n + "/objectName", obj("t", "baja:String", "v", name),
			n + "/objectId", obj("t", "bacnet:BacnetObjectIdentifier", "v", id),
			n + "/description", obj("t", "baja:String", "v", desc)}
		if value != "" {
			kv = append(kv, n+"/value", obj("t", "baja:Double", "v", value))
		}
		return kv
	}
	job := []any{"", obj("h", "9", "t", "bacnet:BacnetDiscoverPointsJob"), "jobState", obj("t", "baja:JobState", "v", state)}
	job = append(job, row("dc0", "SpaceTemp", "analogInput:1", "22.5", "room")...)
	job = append(job, row("dc1", "DamperCmd", "analogOutput:2", "40.0", "damper")...)
	job = append(job, row("dc2", "--", "binaryInput:3", "", "--")...)
	points := []any{"", obj("h", "p1", "t", "bacnet:BacnetPointDeviceExt"),
		"Folder", obj("h", "f1", "t", "bacnet:BacnetPointFolder"),
		"Folder/RoomTemp", obj("h", "q1", "t", "control:NumericPoint"),
		"Folder/RoomTemp/proxyExt", obj("t", "bacnet:BacnetNumericProxyExt"),
		"Folder/RoomTemp/proxyExt/objectId", obj("t", "bacnet:BacnetObjectIdentifier", "v", "analogInput:1")}
	return &loads{nodes: obj(ptsJob, obj(job...), ptsExt, obj(points...),
		ptsDevice, obj("", obj("h", "dv", "t", "bacnet:BacnetDevice"), "points", obj("h", "p1", "t", "bacnet:BacnetPointDeviceExt")))}
}

// The plan invokes the hidden submitPointDiscoveryJob (no argument) on the device's points
// extension, given the device or the extension.
func TestDiscoverPointsPlansTheHiddenJobOnThePointsExt(t *testing.T) {
	for _, ord := range []string{ptsDevice, ptsExt} {
		p, err := writeops.DiscoverPoints.PlanWith(ptsStation("success"), obj("device_ord", ord), clock(), nil)
		if err != nil {
			t.Fatalf("%s: %v", ord, err)
		}
		if got := js(t, p.Ops[0]); got != `{"ssc": "invokeAction", "arg": {"h": "p1", "a": "submitPointDiscoveryJob"}}` {
			t.Errorf("%s: op = %s", ord, got)
		}
		if !strings.Contains(js(t, p.Notes), "no point is added") {
			t.Errorf("notes = %s", js(t, p.Notes))
		}
	}
	c := &loads{nodes: obj(bacnetNet+"/Mod", obj("", obj("h", "m", "t", "modbusTcp:ModbusTcpDevice"), "points", obj("h", "mp", "t", "modbusTcp:ModbusTcpPointDeviceExt")),
		bacnetNet+"/Mod/points", obj("", obj("h", "mp", "t", "modbusTcp:ModbusTcpPointDeviceExt")))}
	if _, err := writeops.DiscoverPoints.PlanWith(c, obj("device_ord", bacnetNet+"/Mod"), clock(), nil); err == nil ||
		!strings.Contains(err.Error(), "BACnet points extension") {
		t.Errorf("non-BACnet: err = %v", err)
	}
	if _, err := writeops.DiscoverPoints.PlanWith(ptsStation("success"), obj("device_ord", ptsDevice, "wait_seconds", 0), clock(), nil); err == nil {
		t.Errorf("wait_seconds 0 accepted")
	}
}

// The read-back lists one row per discovered object, marks objects already proxied under the
// extension (at any depth) and counts rows the station could not read ("--").
func TestDiscoverPointsReadbackRows(t *testing.T) {
	c := ptsStation("success")
	args := obj("device_ord", ptsDevice)
	p, err := writeops.DiscoverPoints.PlanWith(c, args, clock(), nil)
	if err != nil {
		t.Fatal(err)
	}
	reply := obj("t", "baja:Ord", "v", "slot:/Services/JobService/BacnetDiscoverPointsJob2")
	_, _, observed, verdict, err := writeops.DiscoverPoints.Readback(c, args, p, []any{reply}, nil)
	if err != nil || verdict != "verified" {
		t.Fatalf("verdict %q err %v", verdict, err)
	}
	got := js(t, observed)
	for _, want := range []string{
		`"found": 3`, `"unread": 1`,
		`{"object_name": "SpaceTemp", "object_id": "analogInput:1", "value": "22.5", "description": "room", "in_database": "Folder/RoomTemp"}`,
		`{"object_name": "DamperCmd", "object_id": "analogOutput:2", "value": "40.0", "description": "damper", "in_database": null}`,
		`"unread": true`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %s in %s", want, got)
		}
	}
	for state, verdict := range map[string]string{"failed": "mismatch", "running": "unverified"} {
		_, _, o, v, _ := writeops.DiscoverPoints.Readback(ptsStation(state), args, p, []any{reply}, nil)
		if v != verdict {
			t.Errorf("%s: verdict %q observed %s", state, v, js(t, o.(*pyjson.Object)))
		}
	}
}
