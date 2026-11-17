package blackbox

// n4_commissioning_report, the spyder section: the download state of each Honeywell BACnet
// Spyder (deviceStatus=1, compileStatusFlag=0, sErrorStatus empty, enabled true) and, with
// spyder_spec, its IOComp configuration against the caller's table. The rows mirror a client B
// field job: four Spyders after a Full Download, one never finished, one left disabled.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const spyNet = "slot:/Drivers/BacnetNetwork/"

// spyDevices are the Spyder rows: slotPath, name, enabled, deviceStatus, compileStatusFlag, sErrorStatus.
var spyDevices = []string{
	spyNet + "$32005,$32005,true,1,0,",
	spyNet + "$32006,$32006,true,4,4,",
	spyNet + "$32007,$32007,false,1,0,",
	spyNet + "$32009,$32009,true,1,0,Download failed",
}

// spyIO are the IOComp rows: slotPath, SensorType, InputLow, InputHigh, OutputLow, OutputHigh, Terminal1.
var spyIO = []string{
	spyNet + "$32005/ControlProgram/RA_T/IOComp,0,0,0,0,0,1",
	spyNet + "$32005/ControlProgram/COMP1_A/IOComp,8,0,10,0,30,3",
	spyNet + "$32006/ControlProgram/RA_T/IOComp,8,0,10,0,30,1",
}

// spyRig answers the Spyder query with devices and the IOComp query with io (nil: not answered).
func spyRig(t *testing.T, devices, io []string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	routes := []boxstation.OrdRoute{}
	if devices != nil {
		body := "Slot Path,Name,Enabled,Device Status,Compile Status Flag,S Error Status\n" + strings.Join(devices, "\n") + "\n"
		routes = append(routes, boxstation.OrdRoute{Needle: "honeywellBacnetSpyder:BacnetSpyder", Body: []byte(body)})
	}
	if io != nil {
		body := "Slot Path,Sensor Type,Input Low,Input High,Output Low,Output High,Terminal1\n" + strings.Join(io, "\n") + "\n"
		routes = append(routes, boxstation.OrdRoute{Needle: "honeywellSpyderTool:IOComponent", Body: []byte(body)})
	}
	routeOrds(sv, routes...)
	return r, sv
}

func TestCommissioningSpyderFlagsADeviceLeftDisabled(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	c := comWant(t, comSection(t, r, "spyder", nil), "disabled", 1, spyNet+"$32007")
	if fact := comFact(c, spyNet+"$32007"); !strings.Contains(fact, "Enabled=false") || !strings.Contains(fact, "Ping") {
		t.Errorf("fact = %q", fact)
	}
	if item := objects(c["items"])[0]; item["name"] != "2007" {
		t.Errorf("name = %v, want the unescaped slot name", item["name"])
	}
}

func TestCommissioningSpyderFlagsADeviceStatusOtherThanOne(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	c := comWant(t, comSection(t, r, "spyder", nil), "device_status", 1, spyNet+"$32006")
	if fact := comFact(c, spyNet+"$32006"); fact != "deviceStatus 4, want 1" {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningSpyderFlagsACompileStatusFlagOtherThanZero(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	c := comWant(t, comSection(t, r, "spyder", nil), "compile_status", 1, spyNet+"$32006")
	if fact := comFact(c, spyNet+"$32006"); fact != "compileStatusFlag 4, want 0" {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningSpyderFlagsANonEmptyErrorStatus(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	c := comWant(t, comSection(t, r, "spyder", nil), "error_status", 1, spyNet+"$32009")
	if fact := comFact(c, spyNet+"$32009"); fact != `sErrorStatus "Download failed"` {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningSpyderWithoutASpecReportsOnlyTheDownloadState(t *testing.T) {
	r, sv := spyRig(t, spyDevices, spyIO)
	out := r.Srv.OK("n4_commissioning_report", Args{"sections": []any{"spyder"}})
	s := shxSection(t, out, "spyder")
	if why, _ := comCheck(t, s, "io_mismatch")["skipped"].(string); !strings.Contains(why, "spyder_spec") {
		t.Errorf("io_mismatch = %v", comCheck(t, s, "io_mismatch"))
	}
	if s["spyders"] != 4.0 || s["findings"] != 4.0 || comSummary(out, "spyder") != "spyder: 4 finding(s)" {
		t.Errorf("section = %v / %q", s, comSummary(out, "spyder"))
	}
	for _, ord := range ordRequests(sv) {
		if strings.Contains(ord, "IOComponent") {
			t.Errorf("the IOComp table was read without a spec: %s", ord)
		}
	}
}

func TestCommissioningSpyderComparesTheIOCompWithTheSpecTable(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	spec := map[string]any{
		"2005": map[string]any{
			"RA_T":    map[string]any{"sensor_type": 0, "terminal": 1},
			"COMP1_A": map[string]any{"sensor_type": 8, "input_low": 0, "input_high": 10, "output_low": 0, "output_high": 30, "terminal": 4},
			"FAN_A":   map[string]any{"sensor_type": 8},
		},
		"2006": map[string]any{"RA_T": map[string]any{"sensor_type": "0"}},
		"2010": map[string]any{"RA_T": map[string]any{"sensor_type": 0}},
	}
	s := comSection(t, r, "spyder", Args{"spyder_spec": spec})
	comp1, ra6 := spyNet+"$32005/ControlProgram/COMP1_A/IOComp", spyNet+"$32006/ControlProgram/RA_T/IOComp"
	c := comWant(t, s, "io_mismatch", 4, comp1, spyNet+"$32005", ra6, "")
	for path, want := range map[string]string{
		comp1:             "Terminal1 3, want 4",
		spyNet + "$32005": "no point FAN_A with an IOComp below this Spyder",
		ra6:               "SensorType 8, want 0",
		"":                "no Spyder named 2010 in scope",
	} {
		if fact := comFact(c, path); fact != want {
			t.Errorf("%s fact = %q, want %q", path, fact, want)
		}
	}
}

func TestCommissioningSpyderSpecAloneAddsTheSectionToTheDefaultReport(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	if out := r.Srv.OK("n4_commissioning_report", Args{}); out["spyder"] != nil {
		t.Errorf("the default report built the spyder section: %v", out["spyder"])
	}
	out := r.Srv.OK("n4_commissioning_report", Args{"spyder_spec": map[string]any{}})
	if out["spyder"] == nil || out["points"] == nil {
		t.Errorf("spyder_spec without sections: spyder %v, points %v", out["spyder"], out["points"])
	}
}

func TestCommissioningSpyderFailedReadsAreNotChecked(t *testing.T) {
	r, _ := spyRig(t, nil, nil)
	s := comSection(t, r, "spyder", Args{"spyder_spec": map[string]any{"2005": map[string]any{}}})
	for _, name := range []string{"disabled", "device_status", "compile_status", "error_status", "io_mismatch"} {
		if why, _ := comCheck(t, s, name)["not_checked"].(string); why == "" {
			t.Errorf("%s = %v", name, comCheck(t, s, name))
		}
	}
	if s["state"] != "not_checked" {
		t.Errorf("state = %v", s["state"])
	}
	r, _ = spyRig(t, spyDevices, nil) // the IOComp type unknown to the station
	s = comSection(t, r, "spyder", Args{"spyder_spec": map[string]any{"2005": map[string]any{}}})
	if why, _ := comCheck(t, s, "io_mismatch")["not_checked"].(string); !strings.Contains(why, "spyder_io") {
		t.Errorf("io_mismatch = %v", comCheck(t, s, "io_mismatch"))
	}
}

func TestCommissioningSpyderRefusesAMalformedSpec(t *testing.T) {
	r, _ := spyRig(t, spyDevices, spyIO)
	for _, spec := range []map[string]any{
		{"2005": "RA_T"},
		{"2005": map[string]any{"RA_T": 0}},
		{"2005": map[string]any{"RA_T": map[string]any{"sensor": 0}}},
		{"2005": map[string]any{"RA_T": map[string]any{"terminal": []any{1}}}},
	} {
		text := r.Srv.Call("n4_commissioning_report", Args{"sections": []any{"spyder"}, "spyder_spec": spec}).Text
		if !strings.Contains(text, "spyder_spec") {
			t.Errorf("spec %v: %s", spec, text)
		}
	}
}
