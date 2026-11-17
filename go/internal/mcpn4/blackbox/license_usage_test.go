package blackbox

// n4_license_info usage and plan (field retro delta U8): per global capacity rule the devices
// and proxy points the station counts (Metrics.java: the modules the licence excludes are not
// counted) beside device.limit / point.limit, and whether a plan of additions fits.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The type tables of a station with 3 counted devices beside one nrio and one Niagara station
// device, and 4 proxy points.
const (
	usageDevicesCSV = "Type\nbacnet:BacnetDevice\nbacnet:BacnetDevice\nmodbusAsync:ModbusAsyncDevice\n" +
		"nrio:NrioDevice\nniagaraDriver:NiagaraStation\n"
	usagePointsCSV = "Type\nbacnet:BacnetProxyExt\nbacnet:BacnetProxyExt\nmodbusAsync:ModbusAsyncProxyExt\n" +
		"modbusAsync:ModbusAsyncProxyExt\n"
)

// licCapacity is licSmall with a globalCapacity feature holding attrs.
func licCapacity(attrs ...string) []byte {
	lines := []string{"HostId=Lab-0000", "", "Certificates", "  Vendor.certificate [expires: never] {valid}", "",
		"Licenses (perpetual)", "  A.license [expires: never] {valid}", "", "Features", "  tridium:about [expires: never]",
		"    owner=Lab Owner", "  tridium:globalCapacity [expires: never]"}
	for _, a := range attrs {
		lines = append(lines, "    "+a)
	}
	return licPage(append(lines, "", "Brand", "  brandId = LabBrand")...)
}

// routeCounts answers the device and proxy point type selects.
func routeCounts(sv *boxstation.Servlets, devices, points string) {
	routeOrds(sv, boxstation.OrdRoute{Needle: "from driver:Device", Body: []byte(devices)},
		boxstation.OrdRoute{Needle: "from driver:ProxyExt", Body: []byte(points)})
}

var labLimits = []string{"device.limit=26", "point.limit=500", "excludedDevices=ndio;nrio;niagaraDriver"}

func usageRule(t *testing.T, out map[string]any, rule string) map[string]any {
	t.Helper()
	for _, o := range objects(out["usage"]) {
		if o["rule"] == rule {
			return o
		}
	}
	t.Fatalf("no usage rule %q in %v", rule, out["usage"])
	return nil
}

func TestLicenseUsageCountsAgainstTheLimitsLeavingExcludedModulesOut(t *testing.T) {
	r, sv := licRig(t, licNow, licCapacity(labLimits...))
	routeCounts(sv, usageDevicesCSV, usagePointsCSV)
	out := r.Srv.OK("n4_license_info", Args{})
	dev := usageRule(t, out, "devices")
	if dev["used"] != 3.0 || dev["limit"] != 26.0 || dev["headroom"] != 23.0 ||
		!reflect.DeepEqual(dev["excluded_modules"], []any{"ndio", "nrio", "niagaraDriver"}) {
		t.Errorf("devices = %v", dev)
	}
	pts := usageRule(t, out, "points")
	if pts["used"] != 4.0 || pts["limit"] != 500.0 || pts["headroom"] != 496.0 {
		t.Errorf("points = %v", pts)
	}
	if _, ok := out["plan_result"]; ok {
		t.Errorf("plan_result without a plan: %v", out["plan_result"])
	}
}

func TestLicensePlanThatFits(t *testing.T) {
	r, sv := licRig(t, licNow, licCapacity(labLimits...))
	routeCounts(sv, usageDevicesCSV, usagePointsCSV)
	got := r.Srv.OK("n4_license_info", Args{"plan": Args{"add_devices": 23, "add_points": 496}})["plan_result"]
	if !reflect.DeepEqual(got, map[string]any{"fits": true, "over": []any{}}) {
		t.Errorf("plan_result = %v", got)
	}
}

func TestLicensePlanOverByN(t *testing.T) {
	r, sv := licRig(t, licNow, licCapacity(labLimits...))
	routeCounts(sv, usageDevicesCSV, usagePointsCSV)
	got := r.Srv.OK("n4_license_info", Args{"plan": Args{"add_devices": 35, "add_points": 10}})["plan_result"]
	want := map[string]any{"fits": false, "over": []any{map[string]any{"rule": "devices", "by": 12.0}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("plan_result = %v, want %v", got, want)
	}
}

func TestLicenseUnlimitedNeverFailsAPlan(t *testing.T) {
	r, sv := licRig(t, licNow, licCapacity("point.limit=none"))
	routeCounts(sv, usageDevicesCSV, usagePointsCSV)
	out := r.Srv.OK("n4_license_info", Args{"plan": Args{"add_devices": 100000, "add_points": 100000}})
	for _, rule := range []string{"devices", "points"} {
		if u := usageRule(t, out, rule); u["limit"] != nil || u["headroom"] != nil || u["used"] == nil {
			t.Errorf("%s = %v, want a count with limit and headroom null", rule, u)
		}
	}
	if got := out["plan_result"].(map[string]any); got["fits"] != true {
		t.Errorf("plan_result = %v", got)
	}
}

func TestLicenseUnreadableCountIsNullWithANoteAndThePlanUnknown(t *testing.T) {
	r, _ := licRig(t, licNow, licCapacity(labLimits...)) // no BQL route: every select is a 404
	out := r.Srv.OK("n4_license_info", Args{"plan": Args{"add_devices": 1}})
	dev := usageRule(t, out, "devices")
	if dev["used"] != nil || dev["headroom"] != nil || !strings.Contains(dev["note"].(string), "count") {
		t.Errorf("devices = %v, want used null with a note", dev)
	}
	got := out["plan_result"].(map[string]any)
	if got["fits"] != nil || !strings.Contains(got["note"].(string), "unknown") {
		t.Errorf("plan_result = %v, want fits null with a note", got)
	}
}

// The reply without a plan keeps every field it had before usage was added.
func TestLicenseReplyWithoutPlanKeepsItsFields(t *testing.T) {
	r, _ := licRig(t, licNow, licSmall("2027-01-01"))
	out := r.Srv.OK("n4_license_info", Args{})
	keys := []string{}
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	want := []string{"brand", "certificates", "elapsed_ms", "expired", "features", "generated_note", "host_id", "http_status",
		"license_kind", "licenses", "maintenance", "nearest_expiry", "owner", "path", "project", "size_bytes",
		"source", "station_date", "status", "truncated", "usage"}
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
	if out["status"] != "ok" || out["host_id"] != "Lab-0000" {
		t.Errorf("reply = %v", out)
	}
}
