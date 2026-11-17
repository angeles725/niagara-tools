package blackbox

// n4_histories_audit (station-audits S1): one read-only call lists the points with no history,
// the disabled history extensions, the stale histories, the ones close to capacity and the
// history ids shared by several extensions, from a fake station's answers.

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	haDev = "slot:/Plant/Dev1/points/"
	// haService is a control point of a station service (live check 2026-11-17: 52 of 53 such points)
	haService = "slot:/Services/SecurityService/certificates/default/expiry"
)

// haAgo is the encodeToString text of the time d before now (the server reads its own clock).
func haAgo(d time.Duration) string {
	return time.Now().UTC().Add(-d).Format("2006-01-02T15:04:05.000Z")
}

// haTables are the points and history extensions of the fake station, and each history's count.
func haTables() (points, exts []string, counts map[string]string) {
	points = []string{haDev + "Temp", haDev + "NoHist", haDev + "Off", haDev + "Press", haDev + "Door",
		haDev + "Level", haDev + "Amps", "slot:/Plant/Dev2/points/Amps", haService}
	ext := func(point, typ, enabled, id, capacity, policy, interval, last string) string {
		return strings.Join([]string{point + "/hist", "history:" + typ, enabled, id, capacity, policy, interval, last}, ",")
	}
	exts = []string{
		ext(haDev+"Temp", "NumericIntervalHistoryExt", "true", "/FakeStation/Temp", "1:1000", "roll", "false:300000", haAgo(2*time.Minute)),
		ext(haDev+"Off", "NumericIntervalHistoryExt", "false", "", "1:1000", "roll", "false:300000", "null"),
		ext(haDev+"Press", "NumericIntervalHistoryExt", "true", "/FakeStation/Press", "1:1000", "roll", "false:60000", haAgo(2*time.Hour)),
		ext(haDev+"Door", "BooleanCovHistoryExt", "true", "/FakeStation/Door", "1:500", "roll", "true:0", haAgo(48*time.Hour)),
		ext(haDev+"Level", "NumericIntervalHistoryExt", "true", "/FakeStation/Level", "1:1000", "stop", "false:300000", haAgo(time.Minute)),
		ext(haDev+"Amps", "NumericIntervalHistoryExt", "true", "/FakeStation/Amps", "1:1000", "roll", "false:300000", haAgo(time.Minute)),
		ext("slot:/Plant/Dev2/points/Amps", "NumericIntervalHistoryExt", "true", "/FakeStation/Amps", "1:1000", "roll", "false:300000", haAgo(time.Minute)),
	}
	counts = map[string]string{"Temp": "100", "Press": "10", "Door": "20", "Level": "950", "Amps": "10"}
	return points, exts, counts
}

// haRig is a connected rig whose station answers the audit's queries from haTables.
func haRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	points, exts, counts := haTables()
	r, sv := servletRig(t, nil)
	routes := []boxstation.OrdRoute{
		{Needle: "control:ControlPoint", Body: []byte("Slot Path\n" + strings.Join(points, "\n") + "\n")},
		{Needle: "history:HistoryExt", Body: []byte("Slot Path,Type,Enabled,Id,Encode To String,Encode To String," +
			"Encode To String,Encode To String\n" + strings.Join(exts, "\n") + "\n")},
	}
	for name, n := range counts {
		routes = append(routes, boxstation.OrdRoute{Needle: "history:/FakeStation/" + name + "|bql:select count(*)",
			Body: []byte("count(toString)\n" + n + "\n")})
	}
	routeOrds(sv, routes...)
	return r, sv
}

func haPaths(v any, key string) []string {
	out := []string{}
	for _, p := range field(v, key) {
		s, _ := p.(string)
		out = append(out, s)
	}
	return out
}

func TestHistoriesAuditListsEveryFindingOfTheStation(t *testing.T) {
	r, sv := haRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{})
	if got := haPaths(out["no_history"], "point"); !reflect.DeepEqual(got, []string{haDev + "NoHist"}) {
		t.Errorf("no_history = %q", got)
	}
	if got := haPaths(out["disabled"], "ext"); !reflect.DeepEqual(got, []string{haDev + "Off/hist"}) {
		t.Errorf("disabled = %q", got)
	}
	stale := objects(out["stale"])
	if got := haPaths(out["stale"], "history_id"); !reflect.DeepEqual(got, []string{"/FakeStation/Press", "/FakeStation/Door"}) {
		t.Fatalf("stale = %v", out["stale"])
	}
	if stale[0]["interval"] != "1m" || stale[0]["threshold"] != "10m" || stale[1]["interval"] != "cov" || stale[1]["threshold"] != "1d" {
		t.Errorf("stale items = %v", stale)
	}
	near := objects(out["near_capacity"])
	if len(near) != 1 || near[0]["history_id"] != "/FakeStation/Level" || near[0]["records"] != 950.0 ||
		near[0]["capacity"] != 1000.0 || near[0]["full_policy"] != "stop" || near[0]["severity"] != "warning" {
		t.Errorf("near_capacity = %v", near)
	}
	colliding := objects(out["colliding_ids"])
	if len(colliding) != 1 || colliding[0]["history_id"] != "/FakeStation/Amps" ||
		!reflect.DeepEqual(colliding[0]["exts"], []any{haDev + "Amps/hist", "slot:/Plant/Dev2/points/Amps/hist"}) {
		t.Errorf("colliding_ids = %v", colliding)
	}
	counts, _ := out["counts"].(map[string]any)
	want := map[string]any{"points": 8.0, "history_exts": 7.0, "no_history": 1.0, "disabled": 1.0, "stale": 2.0,
		"near_capacity": 1.0, "colliding_ids": 1.0}
	if !reflect.DeepEqual(counts, want) {
		t.Errorf("counts = %v, want %v", counts, want)
	}
	if out["stale_min"] != "10m" || out["services_excluded"] != 1.0 {
		t.Errorf("stale_min %v services_excluded %v", out["stale_min"], out["services_excluded"])
	}
	if out["truncated"] != false || out["count_reads"] != 5.0 || out["requests"] != 7.0 {
		t.Errorf("truncated %v count_reads %v requests %v", out["truncated"], out["count_reads"], out["requests"])
	}
	if n := len(ordRequests(sv)); n != 7 {
		t.Errorf("station requests = %d, want 7 (2 tables, 5 distinct record-capped ids)", n)
	}
}

func TestHistoriesAuditListsServicePointsOnlyWhenAsked(t *testing.T) {
	r, _ := haRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{"exclude_services": false})
	if got := haPaths(out["no_history"], "point"); !reflect.DeepEqual(got, []string{haDev + "NoHist", haService}) {
		t.Errorf("no_history = %q", got)
	}
	if out["services_excluded"] != 0.0 {
		t.Errorf("services_excluded = %v", out["services_excluded"])
	}
}

func TestHistoriesAuditStaleMinIsAFloorUnderTheIntervalThreshold(t *testing.T) {
	r, _ := haRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{"stale_min": "3h"}) // Press: 1 m interval, 2 h old
	if got := haPaths(out["stale"], "history_id"); !reflect.DeepEqual(got, []string{"/FakeStation/Door"}) {
		t.Errorf("stale = %q", got)
	}
}

func TestHistoriesAuditHonoursItsThresholds(t *testing.T) {
	r, _ := haRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{"stale_factor": 200, "cov_max_age": "3d", "capacity_warn": 0.96})
	if s, n := objects(out["stale"]), objects(out["near_capacity"]); len(s) != 0 || len(n) != 0 {
		t.Errorf("stale %v near_capacity %v, want none", s, n)
	}
}

func TestHistoriesAuditDoesNotAssertAbsenceFromACutTable(t *testing.T) {
	r, _ := haRig(t)
	out := r.Srv.OK("n4_histories_audit", Args{"max_histories": 2, "base": "station:|slot:/Plant"})
	counts, _ := out["counts"].(map[string]any)
	if out["truncated"] != true || counts["no_history"] != nil || out["no_history_note"] == nil || counts["history_exts"] != 2.0 {
		t.Errorf("reply = %v", out)
	}
	if out["base"] != "station:|slot:/Plant" {
		t.Errorf("base = %v", out["base"])
	}
}

func TestHistoriesAuditRefusesABadBaseOrSpanBeforeAnyRequest(t *testing.T) {
	r, sv := haRig(t)
	res := r.Srv.Call("n4_histories_audit", Args{"base": "history:/FakeStation/Temp"})
	if !res.IsError || len(ordRequests(sv)) != 0 {
		t.Errorf("bad base: %v, requests %v", res.Raw, ordRequests(sv))
	}
	for _, bad := range []Args{{"cov_max_age": "1w"}, {"stale_min": "0m"}} {
		res = r.Srv.Call("n4_histories_audit", bad)
		if !res.IsError || len(ordRequests(sv)) != 0 {
			t.Errorf("%v: %v, requests %v", bad, res.Raw, ordRequests(sv))
		}
	}
}
