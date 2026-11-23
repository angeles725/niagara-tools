package blackbox

// n4_histories_audit flat_scope and flat_down (retro R6 delta M3, R8 deltas N3 and N4): flat can be limited to
// sensor readings, and a flat history whose point is down goes to flat_down, not to flat.

import (
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	flatInterval = "false:300000"
	flatCOV      = "true:0"
	flatUnitsF   = "units=u:fahrenheit;F;(K);*0.5555555555555556+255.37222222222223;|precision=i:1"
	flatPointsQ  = "status.bits, facets from control:ControlPoint"
)

// flatScopeRig holds four histories that each hold 0.0: Temp (COV, fahrenheit), Mode (COV, no units), Fan
// (interval, no units) and Dead (interval, its point has the down status). pointsRead is false for a station
// that does not answer the point status read.
func flatScopeRig(t *testing.T, pointsRead bool) (*Rig, *boxstation.Servlets) {
	t.Helper()
	names := []string{"Temp", "Mode", "Fan", "Dead"}
	intervals := map[string]string{"Temp": flatCOV, "Mode": flatCOV, "Fan": flatInterval, "Dead": flatInterval}
	var exts, points, status []string
	for _, n := range names {
		exts = append(exts, flatDev+n+"/hist,history:NumericIntervalHistoryExt,true,/FakeStation/"+n+",0:0,roll,"+
			intervals[n]+","+haAgo(time.Minute))
		points = append(points, flatDev+n)
		facets, bits := "precision=i:1", "0"
		if n == "Temp" {
			facets = flatUnitsF
		}
		if n == "Dead" {
			bits = "4"
		}
		status = append(status, flatDev+n+","+bits+","+facets)
	}
	routes := []boxstation.OrdRoute{}
	if pointsRead {
		routes = append(routes, boxstation.OrdRoute{Needle: flatPointsQ,
			Body: []byte("Slot Path,Status Bits,Facets\n" + strings.Join(status, "\n") + "\n")})
	}
	routes = append(routes,
		boxstation.OrdRoute{Needle: "control:ControlPoint", Body: []byte("Slot Path\n" + strings.Join(points, "\n") + "\n")},
		boxstation.OrdRoute{Needle: "history:HistoryExt", Body: []byte("Slot Path,Type,Enabled,Id,Encode To String," +
			"Encode To String,Encode To String,Encode To String\n" + strings.Join(exts, "\n") + "\n")})
	for _, n := range names {
		routes = append(routes,
			boxstation.OrdRoute{Needle: "history:/FakeStation/" + n + "?start=", Body: []byte("min(value),max(value)\n0.0,0.0\n")},
			boxstation.OrdRoute{Needle: "history:/FakeStation/" + n + "?end=", Body: []byte("count(toString)\n600\n")})
	}
	r, sv := servletRig(t, nil)
	routeOrds(sv, routes...)
	return r, sv
}

func flatIDs(out map[string]any, list string) []string {
	ids := []string{}
	for _, f := range objects(out[list]) {
		ids = append(ids, strings.TrimPrefix(f["history_id"].(string), "/FakeStation/"))
	}
	return ids
}

func sameIDs(got []string, want ...string) bool {
	return strings.Join(got, ",") == strings.Join(want, ",")
}

func TestHistoriesAuditFlatWithoutScopeOrDownKeepsEveryFlatHistory(t *testing.T) {
	r, sv := flatScopeRig(t, true)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true})
	if ids := flatIDs(out, "flat"); !sameIDs(ids, "Temp", "Mode", "Fan", "Dead") {
		t.Errorf("flat = %v", ids)
	}
	for _, k := range []string{"flat_down", "flat_scope"} {
		if _, has := out[k]; has {
			t.Errorf("%s present by default: %v", k, out[k])
		}
	}
	for _, req := range ordRequests(sv) {
		if strings.Contains(req, flatPointsQ) {
			t.Errorf("the point status was read without flat_scope or flat_down: %s", req)
		}
	}
	if out["requests"] != 10.0 {
		t.Errorf("requests %v", out["requests"])
	}
}

func TestHistoriesAuditFlatScopeSensorsKeepsIntervalHistoriesAndUnits(t *testing.T) {
	r, sv := flatScopeRig(t, true)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true, "flat_scope": "sensors"})
	if ids := flatIDs(out, "flat"); !sameIDs(ids, "Temp", "Fan", "Dead") || out["flat_scope"] != "sensors" {
		t.Errorf("flat = %v flat_scope %v", ids, out["flat_scope"])
	}
	for _, req := range ordRequests(sv) {
		if strings.Contains(req, "Mode") && strings.Contains(req, "history:") {
			t.Errorf("a setting history was read: %s", req)
		}
	}
	// the two base reads, one point read, then a window read and a count for each of the three sensors
	if out["requests"] != 9.0 || out["flat_reads"] != 7.0 {
		t.Errorf("requests %v flat_reads %v", out["requests"], out["flat_reads"])
	}
}

func TestHistoriesAuditFlatDownSeparatesAHistoryWhosePointIsDown(t *testing.T) {
	r, _ := flatScopeRig(t, true)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true, "flat_down": true})
	if ids := flatIDs(out, "flat"); !sameIDs(ids, "Temp", "Mode", "Fan") {
		t.Errorf("flat = %v", ids)
	}
	down := objects(out["flat_down"])
	if len(down) != 1 || down[0]["history_id"] != "/FakeStation/Dead" || down[0]["point"] != flatDev+"Dead" ||
		down[0]["value"] != "0.0" || !strings.Contains(down[0]["note"].(string), "device down, not a placeholder") {
		t.Fatalf("flat_down = %v", out["flat_down"])
	}
	if flags, _ := down[0]["status"].([]any); len(flags) != 1 || flags[0] != "down" {
		t.Errorf("status = %v", down[0]["status"])
	}
	counts, _ := out["counts"].(map[string]any)
	if counts["flat"] != 3.0 || counts["flat_down"] != 1.0 {
		t.Errorf("counts %v", counts)
	}
}

func TestHistoriesAuditFlatDownAndScopeTogether(t *testing.T) {
	r, _ := flatScopeRig(t, true)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true, "flat_down": true, "flat_scope": "sensors"})
	if f, d := flatIDs(out, "flat"), flatIDs(out, "flat_down"); !sameIDs(f, "Temp", "Fan") || !sameIDs(d, "Dead") {
		t.Errorf("flat %v flat_down %v", f, d)
	}
}

func TestHistoriesAuditFlatKeepsEveryHistoryWhenThePointReadFails(t *testing.T) {
	r, _ := flatScopeRig(t, false)
	out := r.Srv.OK("n4_histories_audit", Args{"flat": true, "flat_down": true, "flat_scope": "sensors"})
	if ids := flatIDs(out, "flat"); !sameIDs(ids, "Temp", "Mode", "Fan", "Dead") {
		t.Errorf("flat = %v", ids)
	}
	if ids := flatIDs(out, "flat_down"); len(ids) != 0 {
		t.Errorf("flat_down = %v", ids)
	}
	notes, _ := out["notes"].([]any)
	found := false
	for _, n := range notes {
		found = found || strings.HasPrefix(n.(string), "the flat point read failed, so flat_scope and flat_down were not applied: ")
	}
	if !found {
		t.Errorf("notes = %v", notes)
	}
}
