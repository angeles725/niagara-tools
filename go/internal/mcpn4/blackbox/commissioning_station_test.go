package blackbox

// n4_commissioning_report: the station section, the bounds, and what a failed read says.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// comStationRig is a report rig whose station has a clock, a licence page with the expiries
// given and two services, one of them in fault.
func comStationRig(t *testing.T, expiries ...string) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := comRig(t, nil)
	shxSystemService(r.Station, map[string][2]string{"systemTime": {"baja:AbsTime", licNow}})
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall(expiries...)})
	r.Station.Do(func(st *boxstation.Station) {
		services := comServices(st)
		weather := st.Component(services, "WeatherService", "weather:WeatherService", "")
		weather.Children = append(weather.Children, &boxstation.Node{Name: "status", Type: "baja:Status", Value: "2"},
			&boxstation.Node{Name: "faultCause", Type: "baja:String", Value: "Unlicensed"})
		st.Component(services, "UserService", "baja:UserService", "")
	})
	return r, sv
}

func TestCommissioningChecksTheStationOnlyWhenTheScopeIsTheWholeStation(t *testing.T) {
	r, _ := comStationRig(t, "never")
	out := r.Srv.OK("n4_commissioning_report", Args{"scope": Folder, "sections": []any{"station"}})
	if s := shxSection(t, out, "station"); s["state"] != "skipped" || !strings.Contains(comSummary(out, "station"), "skipped: the scope is not the whole station") {
		t.Errorf("subtree scope: %v / %q", s, comSummary(out, "station"))
	}
	out = r.Srv.OK("n4_commissioning_report", Args{"sections": []any{"station"}})
	s := shxSection(t, out, "station")
	for _, name := range []string{"unsaved_changes", "last_backup"} { // never guessed
		if why, _ := comCheck(t, s, name)["not_checked"].(string); why == "" {
			t.Errorf("%s = %v", name, comCheck(t, s, name))
		}
	}
	if !strings.HasPrefix(comSummary(out, "station"), "station: 1 finding(s), 2 check(s) not checked") {
		t.Errorf("summary = %q", comSummary(out, "station"))
	}
}

func TestCommissioningListsTheServicesInFault(t *testing.T) {
	r, _ := comStationRig(t, "never")
	s := comSection(t, r, "station", nil)
	path := "slot:/Services/WeatherService"
	c := comWant(t, s, "services_in_fault", 1, path)
	if fact := comFact(c, path); fact != "status fault: Unlicensed" || c["checked"] != 4.0 {
		t.Errorf("fact = %q, checked %v", fact, c["checked"])
	}
}

func TestCommissioningWarnsOfALicenceExpiringSoon(t *testing.T) {
	r, sv := comStationRig(t, "2026-12-15") // 30 days from the station date
	c := comCheck(t, comSection(t, r, "station", nil), "licence_expiry")
	if items := objects(c["items"]); c["count"] != 1.0 || !strings.Contains(items[0]["fact"].(string), "30 day(s)") {
		t.Errorf("licence_expiry = %v", c)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("2027-03-31")})
	if c = comCheck(t, comSection(t, r, "station", nil), "licence_expiry"); c["count"] != 0.0 {
		t.Errorf("far expiry: %v", c)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Status: 403})
	if why, _ := comCheck(t, comSection(t, r, "station", nil), "licence_expiry")["not_checked"].(string); why == "" {
		t.Error("an unreadable licence page is reported as checked")
	}
}

func TestCommissioningSaysTruncatedAtTheComponentCapAndAssertsNoAbsence(t *testing.T) {
	r, _ := comRig(t, comHistoryTables)
	out := r.Srv.OK("n4_commissioning_report", Args{"sections": []any{"histories"}, "max_components": 1})
	s := shxSection(t, out, "histories")
	want := "truncated at 1 components: counts are lower bounds and absences are not asserted; narrow the scope"
	if out["truncated"] != true || out["truncated_note"] != want || s["truncated"] != true ||
		!strings.HasSuffix(comSummary(out, "histories"), "(truncated at 1 components, narrow the scope)") {
		t.Errorf("truncated %v, note %v, summary %q", out["truncated"], out["truncated_note"], comSummary(out, "histories"))
	}
	if why, _ := comCheck(t, s, "points_without_history")["not_checked"].(string); !strings.Contains(why, "truncated") {
		t.Errorf("points_without_history = %v", comCheck(t, s, "points_without_history"))
	}
}

func TestCommissioningNeverReportsAFailedQueryAsClean(t *testing.T) {
	r, _ := comRig(t, map[string][]string{"points": {comFailed}, "histories": {comFailed}})
	out := r.Srv.OK("n4_commissioning_report", Args{"sections": []any{"points", "histories", "naming"}})
	for _, name := range []string{"points", "histories", "naming"} {
		s := shxSection(t, out, name)
		if s["state"] != "not_checked" || s["findings"] != nil || !strings.HasPrefix(comSummary(out, name), name+": not checked: ") ||
			!strings.Contains(comSummary(out, name), "404") {
			t.Errorf("%s = %v / %q", name, s, comSummary(out, name))
		}
		for check, v := range s["checks"].(map[string]any) {
			if c := v.(map[string]any); c["not_checked"] == nil || c["count"] != nil {
				t.Errorf("%s.%s = %v", name, check, c)
			}
		}
	}
}

func TestCommissioningPutsTheSummaryFirstAndStatesTheStationRequests(t *testing.T) {
	r, sv := comStationRig(t, "never")
	comAlarmService(r, "defaultAlarmClass")
	gets, posts := getCount(sv), r.Station.Requests()
	res := r.Srv.Call("n4_commissioning_report", nil)
	out := res.Structured
	if res.IsError || !strings.HasPrefix(res.Text, `{"scope": "station:|slot:/", "summary": [`) {
		t.Fatalf("reply starts %.80s", res.Text)
	}
	var names []string
	for _, line := range out["summary"].([]any) {
		name, _, _ := strings.Cut(line.(string), ": ")
		names = append(names, name)
	}
	if !reflect.DeepEqual(names, []string{"points", "histories", "alarms", "links", "naming", "station"}) {
		t.Errorf("summary = %v", out["summary"])
	}
	// 7 GETs (networks, devices, points, history extensions, alarm extensions, alarm classes, the
	// licence page) and 4 slot loads (AlarmService, the link scan, Services, SystemService)
	if out["requests"] != 11.0 || getCount(sv)-gets != 7 || r.Station.Requests()-posts < 4 {
		t.Errorf("requests %v, %d GETs, %d POSTs", out["requests"], getCount(sv)-gets, r.Station.Requests()-posts)
	}
	gets = getCount(sv)
	out = r.Srv.OK("n4_commissioning_report", Args{"sections": []any{"points", "histories", "naming"}})
	if out["requests"] != 4.0 || getCount(sv)-gets != 4 || out["links"] != nil {
		t.Errorf("three sections: requests %v, %d GETs", out["requests"], getCount(sv)-gets)
	}
}

func TestCommissioningBoundsEveryListAndRefusesAnUnknownSection(t *testing.T) {
	r, _ := comRig(t, comNamingTables)
	c := comCheck(t, comSection(t, r, "naming", Args{"limit": 1}), "palette_default_names")
	if c["count"] != 4.0 || len(objects(c["items"])) != 1 || c["more"] != 3.0 {
		t.Errorf("limit 1: %v", c)
	}
	if code, msg := rpcRefusal(t, r.Srv, "n4_commissioning_report", Args{"sections": []any{"points", "wiring"}}); code != -32602 ||
		msg != "argument sections[1] must be one of points, histories, alarms, links, naming, station, spyder; got 'wiring'" {
		t.Errorf("unknown section: %d %s", code, msg)
	}
	if msg := r.Srv.Err("n4_commissioning_report", Args{"scope": "Drivers"}); !strings.Contains(msg, "slot path") {
		t.Errorf("bad scope: %s", msg)
	}
}
