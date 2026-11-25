package readtools_test

// n4_commissioning_report, spyder_hardware (issue #820): the spyder section's model_mismatch check: each Honeywell BACnet Spyder's
// configured model against the hardware id its analogValue:0 point reads. The hardware id lives only
// on the controller (BBacnetSpyder.readControllerModelFromDevice reads analogValue:0 presentValue,
// BBacnetSpyder.java:2871-2880), so the only station-side copy is a point proxying that object.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const modelNet = "slot:/Drivers/BacnetNetwork/"

// modelServer answers the download state of the Spyders (all good), their configured model
// (slotPath, name, controllerModel; nil: the query is not answered) and the numeric points that
// proxy a BACnet object (slotPath, proxyExt.objectId, out.value).
func modelServer(t *testing.T, devices, points []string) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	state := "Slot Path,Name,Enabled,Device Status,Compile Status Flag,S Error Status\n"
	for _, d := range modelDevices {
		path, name, _ := strings.Cut(d, ",")
		state += path + "," + strings.Split(name, ",")[0] + ",true,1,0,\n"
	}
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "sErrorStatus from honeywellBacnetSpyder", Body: state}}
	if devices != nil {
		s.Station.OrdRoutes = append(s.Station.OrdRoutes, readtoolstest.OrdRoute{Needle: "controllerModel from honeywellBacnetSpyder",
			Body: "Slot Path,Name,Controller Model\n" + strings.Join(devices, "\n") + "\n"},
			readtoolstest.OrdRoute{Needle: "proxyExt.objectId", Body: "Slot Path,Object Id,Value\n" + strings.Join(points, "\n") + "\n"})
	}
	return s
}

var (
	modelDevices = []string{
		modelNet + "$32005,$32005,ModelMicroBACnet4", // PUB1012S
		modelNet + "$32006,$32006,ModelBACnet3",      // PUB6438S
		modelNet + "$32007,$32007,ModelBACnet3",
		modelNet + "$32008,$32008,ModelBACnet3",
		modelNet + "$32009,$32009,ModelBACnet3",
	}
	modelPoints = []string{
		modelNet + "$32005/points/AV0MObjHardwareI,analogValue:0,54.0 {ok}",
		modelNet + "$32006/points/AV_HardwareID,analogValue:0,54.0",
		modelNet + "$32006/points/Room_T,analogValue:1,21.5",
		// $32007 proxies no analogValue:0
		modelNet + "$32008/points/AV_HardwareID,analogValue:0,99.0",
		modelNet + "$32009/points/AV_HardwareID,analogValue:0,null",
		"slot:/Drivers/ModbusAsyncNetwork/Gw/points/Hw,analogValue:0,33.0", // below no Spyder
	}
)

func modelSection(t *testing.T, s *readtoolstest.Server) *pyjson.Object {
	t.Helper()
	out := callOK(t, s, "n4_commissioning_report", "sections", []any{"spyder"}, "spyder_hardware", true)
	return field(out, "spyder").(*pyjson.Object)
}

// A Spyder configured for one model whose analogValue:0 reads another is a finding; a matching one,
// one without the point, an unknown id and an unreadable value are counted and not findings.
func TestCommissioningSpyderModelFlagsAConfiguredModelThatTheHardwareContradicts(t *testing.T) {
	s := modelServer(t, modelDevices, modelPoints)
	sec := modelSection(t, s)
	check := field(field(sec, "checks").(*pyjson.Object), "model_mismatch").(*pyjson.Object)
	items := field(check, "items").([]any)
	if field(check, "count") != 2 || len(items) != 2 {
		t.Fatalf("model_mismatch = %s", pyjson.Repr(check))
	}
	got := map[string]string{}
	for _, it := range items {
		got[field(it.(*pyjson.Object), "path").(string)] = field(it.(*pyjson.Object), "fact").(string)
	}
	for path, want := range map[string]string{
		modelNet + "$32006": "configured model PUB6438S (ModelBACnet3), but analogValue:0 reads hardware id 54: PUB1012S (ModelMicroBACnet4)",
		modelNet + "$32008": "configured model PUB6438S (ModelBACnet3), but analogValue:0 reads hardware id 99, which is in no known model",
	} {
		if got[path] != want {
			t.Errorf("%s fact = %q, want %q", path, got[path], want)
		}
	}
	for k, want := range map[string]any{"spyders": 5, "with_hardware_point": 4, "hardware_unreadable": 1} {
		if field(sec, k) != want {
			t.Errorf("%s = %v, want %v", k, field(sec, k), want)
		}
	}
	if field(sec, "state") != "findings" {
		t.Errorf("state = %v", field(sec, "state"))
	}
}

// A read that fails is not a clean check: model_mismatch says why and the download-state checks still ran.
func TestCommissioningSpyderModelFailedReadIsNotChecked(t *testing.T) {
	s := modelServer(t, nil, nil)
	sec := modelSection(t, s)
	check := field(field(sec, "checks").(*pyjson.Object), "model_mismatch").(*pyjson.Object)
	if why, _ := field(check, "not_checked").(string); !strings.Contains(why, "spyder_model") || field(sec, "state") != "partial" {
		t.Errorf("section = %s", pyjson.Repr(sec))
	}
}

// Without spyder_hardware the check is absent and its queries are not sent, whether the section
// comes from sections or from spyder_spec; spyder_hardware alone adds the spyder section.
func TestCommissioningSpyderModelIsOnlyCheckedWhenAsked(t *testing.T) {
	s := modelServer(t, modelDevices, modelPoints)
	for _, kv := range [][]any{{"sections", []any{"spyder"}}, {"spyder_spec", pyjson.NewObject()}} {
		out := callOK(t, s, "n4_commissioning_report", kv...)
		sec, _ := field(out, "spyder").(*pyjson.Object)
		if sec == nil || field(field(sec, "checks").(*pyjson.Object), "model_mismatch") != nil {
			t.Errorf("%v: spyder section %v", kv[0], sec)
		}
	}
	for _, ord := range sent(s) {
		if strings.Contains(ord, "proxyExt.objectId") || strings.Contains(ord, "controllerModel") {
			t.Errorf("a model query was sent unasked: %s", ord)
		}
	}
	if out := callOK(t, s, "n4_commissioning_report", "spyder_hardware", true); field(out, "spyder") == nil {
		t.Errorf("spyder_hardware alone did not add the spyder section")
	}
}
