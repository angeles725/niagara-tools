package blackbox

// The dry run of n4_create_component refuses a device (a driver:Device subtype) whose creation
// would take the counted devices over the licence's globalCapacity device.limit (field retro
// delta U8): a device over the limit is added but sits in fault at start.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

var createDevice = Args{"parent_ord": Folder, "name": "Ahu9", "type": "bacnet:BacnetDevice"}

// licWriteRig is a verified writes-allowed rig whose station serves licence as its licence
// manager page and devices as its device type table, and knows bacnet:BacnetDevice as a device.
func licWriteRig(t *testing.T, licence []byte, devices string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	var sv *boxstation.Servlets
	r := NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		sv = boxstation.ServeServlets(t, r.Station)
		o.Stations = map[string]string{StationName: sv.URL}
	})
	r.ConnectVerified()
	r.Station.SetSupertype("bacnet:BacnetDevice", "driver:Device")
	if licence != nil {
		shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licence})
	}
	if devices != "" {
		routeCounts(sv, devices, usagePointsCSV)
	}
	return r, sv
}

// devicesCSV is a device type table of n counted devices and one nrio device.
func devicesCSV(n int) string {
	return "Type\n" + strings.Repeat("bacnet:BacnetDevice\n", n) + "nrio:NrioDevice\n"
}

func TestCreateOfADeviceOverTheLicenceLimitIsRefused(t *testing.T) {
	r, _ := licWriteRig(t, licCapacity(labLimits...), devicesCSV(26))
	refusedDry(t, r, "n4_create_component", createDevice,
		"bacnet:BacnetDevice", "device.limit 26", "already counts 26 device(s)", "ndio, nrio, niagaraDriver")
	if len(r.Children()) != 0 {
		t.Errorf("children %v", r.Children())
	}
}

func TestCreateOfADeviceUnderTheLicenceLimitIsPlanned(t *testing.T) {
	r, _ := licWriteRig(t, licCapacity(labLimits...), devicesCSV(25))
	plan := r.Srv.Dry("n4_create_component", createDevice)
	if tok, _ := plan["confirmation_token"].(string); tok == "" {
		t.Fatalf("no token: %v", plan)
	}
	if strings.Contains(pyjsonText(plan["plan"]), "licence") {
		t.Errorf("a create within the limit carries a licence note: %v", plan["plan"])
	}
	r.Srv.OK("n4_create_component", Execute(createDevice, plan["confirmation_token"]))
	if children := r.Children(); len(children) != 1 || children[0] != "Ahu9" {
		t.Errorf("children %v", children)
	}
}

func TestCreateOfAnExcludedModuleDeviceIsNotCounted(t *testing.T) {
	r, _ := licWriteRig(t, licCapacity(labLimits...), devicesCSV(26))
	r.Station.SetSupertype("nrio:NrioDevice", "driver:Device")
	plan := r.Srv.Dry("n4_create_component", withArgs(createDevice, Args{"type": "nrio:NrioDevice"}))
	if tok, _ := plan["confirmation_token"].(string); tok == "" {
		t.Fatalf("an excluded module's device was refused: %v", plan)
	}
}

func TestCreateOfADeviceWithAnUnreadableLicenceIsPlannedWithANote(t *testing.T) {
	r, _ := licWriteRig(t, nil, "") // no licence page: the spy read is a 404
	plan := r.Srv.Dry("n4_create_component", createDevice)
	if tok, _ := plan["confirmation_token"].(string); tok == "" {
		t.Fatalf("no token: %v", plan)
	}
	if !strings.Contains(pyjsonText(plan["plan"]), "licence device limit was not checked") {
		t.Errorf("plan lacks the unchecked note: %v", plan["plan"])
	}
}

func TestCreateOfANonDeviceTypeReadsNoLicence(t *testing.T) {
	r, sv := licWriteRig(t, licCapacity("device.limit=0"), devicesCSV(5))
	plan := r.Srv.Dry("n4_create_component", createPump)
	if tok, _ := plan["confirmation_token"].(string); tok == "" {
		t.Fatalf("no token: %v", plan)
	}
	if spy, ords := shxSpyRequests(sv), ordRequests(sv); len(spy) != 0 || len(ords) != 0 {
		t.Errorf("a non-device create read spy %v and ords %v", spy, ords)
	}
}
