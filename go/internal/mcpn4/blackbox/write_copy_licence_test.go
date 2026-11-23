package blackbox

// The dry runs of n4_copy_component and n4_replicate_points refuse a plan that would take the
// counted devices or proxy points over the licence's globalCapacity limit (retro R12 ΔM3): the
// copies would be added but sit in fault at start.

import (
	"strings"
	"testing"
)

const proxyExt = "bacnet:BacnetProxyExt"

func pointsCSV(n int) string { return "Type\n" + strings.Repeat(proxyExt+"\n", n) }

// copyRig is a licence rig holding the device Ahu1 and the point K1 with a proxy extension, both
// under the Folder, and the empty folders A and B.
func copyRig(t *testing.T, devices, points int) *Rig {
	r, sv := licWriteRig(t, licCapacity(labLimits...), devicesCSV(devices))
	routeCounts(sv, devicesCSV(devices), pointsCSV(points))
	r.Station.SetSupertype(proxyExt, "driver:ProxyExt")
	r.Add("3", "Ahu1", "bacnet:BacnetDevice")
	k := r.Add("3", "K1", "")
	r.Add(k.Handle, "proxyExt", proxyExt)
	r.Add("3", "A", "baja:Folder")
	r.Add("3", "B", "baja:Folder")
	return r
}

func copyDevice(names ...any) Args {
	return Args{"parent_ord": Folder, "name": "Ahu1", "new_names": names}
}

func replicateK1() Args {
	return Args{"source_parent_ord": Folder, "points": []any{"K1"}, "targets": []any{
		map[string]any{"parent_ord": Folder + "/A"}, map[string]any{"parent_ord": Folder + "/B"}}}
}

func TestCopyOfDevicesOverTheLicenceLimitIsRefused(t *testing.T) {
	r := copyRig(t, 25, 0)
	refusedDry(t, r, "n4_copy_component", copyDevice("Ahu2", "Ahu3"),
		"this copy would add 2 devices: the station already counts 25/26 devices (licence tridium:globalCapacity "+
			"device.limit), so the new ones would sit in fault at start (modules not counted: ndio, nrio, niagaraDriver); "+
			"free some or raise the limit first (n4_license_info shows the usage)")
	plan := r.Srv.Dry("n4_copy_component", copyDevice("Ahu2"))
	if tok, _ := plan["confirmation_token"].(string); tok == "" || strings.Contains(pyjsonText(plan["plan"]), "licence") {
		t.Errorf("a copy within the limit: %v", plan)
	}
}

func TestReplicationOverTheProxyPointLimitIsRefused(t *testing.T) {
	r := copyRig(t, 0, 499)
	refusedDry(t, r, "n4_replicate_points", replicateK1(),
		"this replication would add 2 points: the station already counts 499/500 points (licence tridium:globalCapacity "+
			"point.limit), so the new ones would sit in fault at start (modules not counted: none); "+
			"free some or raise the limit first (n4_license_info shows the usage)")
	r = copyRig(t, 0, 498)
	if tok, _ := r.Srv.Dry("n4_replicate_points", replicateK1())["confirmation_token"].(string); tok == "" {
		t.Error("a replication that fits the limit was refused")
	}
}

func TestCopyOfAPointOverTheProxyPointLimitIsRefused(t *testing.T) {
	r := copyRig(t, 0, 500)
	refusedDry(t, r, "n4_copy_component", Args{"parent_ord": Folder, "name": "K1", "new_names": []any{"K2"}},
		"this copy would add 1 points: the station already counts 500/500 points")
}

// A licence that cannot be read leaves the dry run of a copy as it was: no refusal, no note.
func TestCopyWithAnUnreadableLicenceIsPlannedAsBefore(t *testing.T) {
	r, _ := licWriteRig(t, nil, "") // no licence page: the spy read is a 404
	r.Add("3", "Ahu1", "bacnet:BacnetDevice")
	plan := r.Srv.Dry("n4_copy_component", copyDevice("Ahu2"))
	if tok, _ := plan["confirmation_token"].(string); tok == "" || strings.Contains(pyjsonText(plan["plan"]), "icence") {
		t.Errorf("plan = %v", plan)
	}
}
