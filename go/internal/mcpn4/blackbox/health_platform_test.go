package blackbox

// n4_station_health platform=true: platform capacity and what is about to expire (S13).

import (
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func pfPlatform(t *testing.T, r *Rig, args Args) map[string]any {
	t.Helper()
	return shxSection(t, r.Srv.OK("n4_station_health", args), "platform")
}

func TestStationHealthPlatformReportsCapacityAndExpiries(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	p := pfPlatform(t, r, Args{"platform": true})
	// sysinfo.html: totalMemory 1000960kb, freeMemory 611941kb; the G1 Old Gen maximum 2097152 KB
	wantHeap := map[string]any{"used_kb": 389019.0, "committed_kb": 1000960.0, "max_kb": 2097152.0, "fraction": 0.185,
		"source": "spy:/sysInfo, spy:/util/JMX Info"}
	if !reflect.DeepEqual(p["heap"], wantHeap) {
		t.Errorf("heap = %v", p["heap"])
	}
	if want := map[string]any{"current": 85.0, "overall": 65.0, "cpus": 16.0}; !reflect.DeepEqual(p["cpu"], want) {
		t.Errorf("cpu = %v", p["cpu"])
	}
	if want := map[string]any{"total": 8000.0, "free": 5000.0}; !reflect.DeepEqual(p["memory"], want) {
		t.Errorf("memory = %v", p["memory"])
	}
	hdb := p["history_db"].(map[string]any)
	if hdb["histories"] != 2.0 || hdb["records"] != 350.0 || hdb["bytes"] != nil || !strings.Contains(hdb["bytes_note"].(string), "record counts") {
		t.Errorf("history_db = %v", hdb)
	}
	lic := p["licence"].(map[string]any)
	if lic["expires"] != "2027-03-31" || lic["days_left"] != 136.0 || lic["never_expires"] != false {
		t.Errorf("licence = %v", lic)
	}
	certs := objects(p["certificates"])
	var got []string
	for _, c := range certs {
		got = append(got, c["alias"].(string)+"|"+c["store"].(string)+"|"+c["expires"].(string))
	}
	want := []string{"tlsrenew|station|2026-11-25", "default|station|2027-10-10", "oldvendor|user|2026-01-05",
		"labserverca|user|2026-12-01", "oldroot2 [jdk]|system|2019-07-09", "oldroot1 [jdk]|system|2020-05-30",
		"oldroot3 [jdk]|system|2026-01-05", "labrootca|system|2035-03-31"}
	if !reflect.DeepEqual(got, want) || certs[0]["days_left"] != 10.0 || certs[1]["days_left"] != 329.0 ||
		certs[1]["status"] != "ok" || certs[2]["days_left"] != -314.0 || certs[3]["days_left"] != 16.0 {
		t.Errorf("certificates = %v", certs)
	}
	if !pfWarned(p, "labserverca", "16 day") || !pfWarned(p, "oldvendor", "expired") || pfWarned(p, "labrootca") ||
		!pfWarned(p, "tlsrenew", "station", "10 day") || pfWarned(p, "default") {
		t.Errorf("warnings = %v", p["warnings"])
	}
	// the expired JDK roots: one summary line, never one warning each
	if !pfWarned(p, "3 system trust-store certificates expired or expiring (JDK roots; informational)") ||
		pfWarned(p, "oldroot1") || len(p["warnings"].([]any)) != 4 {
		t.Errorf("warnings = %v", p["warnings"])
	}
	if pfWarned(p, "licence") { // 136 days left: outside the default 30-day window
		t.Errorf("warnings = %v", p["warnings"])
	}
	mods := p["modules"].(map[string]any)
	if mods["count"] != 6.0 || mods["truncated"] != false || mods["items"] != nil {
		t.Errorf("modules = %v", mods)
	}
	if note := p["certificates_note"].(string); !strings.Contains(note, "trust stores") || !strings.Contains(note, "CertificateExpiryPoint") {
		t.Errorf("certificates_note = %v", p["certificates_note"])
	}
}

func TestStationHealthPlatformLicenceWithNoExpiry(t *testing.T) {
	r, _ := pfRig(t, licSmall("never", "never"))
	p := pfPlatform(t, r, Args{"platform": true})
	want := map[string]any{"expires": nil, "days_left": nil, "never_expires": true}
	if !reflect.DeepEqual(p["licence"], want) || pfWarned(p, "licence") {
		t.Errorf("licence = %v, warnings %v", p["licence"], p["warnings"])
	}
}

func TestStationHealthPlatformWarnsOfAFullHeapAndANearLicence(t *testing.T) {
	r, sv := pfRig(t, licRecorded(t))
	shxServeSpy(sv, pfJmxPath, boxstation.SpyPage{Body: pfJmxPage("400000")})
	p := pfPlatform(t, r, Args{"platform": true, "expiry_warn_days": 200})
	if p["heap"].(map[string]any)["fraction"] != 0.973 || !pfWarned(p, "heap", "97.3%") {
		t.Errorf("heap %v, warnings %v", p["heap"], p["warnings"])
	}
	if !pfWarned(p, "licence", "2027-03-31", "136 day") {
		t.Errorf("warnings = %v", p["warnings"])
	}
}

func TestStationHealthPlatformWarnWindowIsAnArgument(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	p := pfPlatform(t, r, Args{"platform": true, "expiry_warn_days": 10})
	if pfWarned(p, "labserverca") || !pfWarned(p, "oldvendor", "expired") {
		t.Errorf("warnings = %v", p["warnings"])
	}
}

// A page that does not answer leaves its item null with a note saying why, never a guess; the
// others still answer.
func TestStationHealthPlatformUnavailableSourcesAreNullWithANote(t *testing.T) {
	r, sv := pfRig(t, licRecorded(t))
	sv.Do(func(sv *boxstation.Servlets) { delete(sv.SpyPages, pfJmxPath) })
	shxServeSpy(sv, pfHistoryPath, boxstation.SpyPage{Status: 403})
	shxServeSpy(sv, pfCryptoPath, boxstation.SpyPage{Status: 403})
	p := pfPlatform(t, r, Args{"platform": true})
	heap := p["heap"].(map[string]any)
	if heap["used_kb"] != 389019.0 || heap["max_kb"] != nil || heap["fraction"] != nil ||
		!strings.Contains(heap["max_note"].(string), "not_found") {
		t.Errorf("heap = %v", heap)
	}
	if v, present := p["history_db"]; !present || v != nil || !strings.Contains(p["history_db_note"].(string), "403") {
		t.Errorf("history_db = %v, note %v", v, p["history_db_note"])
	}
	// the trust stores are unread; the station's own certificates still answer
	certs := objects(p["certificates"])
	if len(certs) != 2 || certs[0]["store"] != "station" || certs[1]["store"] != "station" ||
		!strings.Contains(p["certificates_note"].(string), "403") {
		t.Errorf("certificates = %v, note %v", certs, p["certificates_note"])
	}
	if p["licence"] == nil || p["cpu"] == nil {
		t.Errorf("the other items must still answer: %v", p)
	}
}

func TestStationHealthPlatformListsModulesWhenAsked(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	mods := pfPlatform(t, r, Args{"platform": true, "include_modules": true})["modules"].(map[string]any)
	items := objects(mods["items"])
	if mods["count"] != 6.0 || len(items) != 6 ||
		!reflect.DeepEqual(items[2], map[string]any{"name": "alarm-rt", "vendor": "Tridium", "version": "4.15.6.20"}) {
		t.Errorf("modules = %v", mods)
	}
}

// Without platform (or platform=false) the reply is the one the tool gave before it, and none
// of the platform pages is read.
func TestStationHealthWithoutPlatformKeepsItsReply(t *testing.T) {
	want := []string{"capacity", "capacity_truncated", "devices", "elapsed_ms", "license", "networks", "points", "system", "totals"}
	for _, args := range []Args{nil, {"platform": false}} {
		r, sv := pfRig(t, licRecorded(t))
		out := r.Srv.OK("n4_station_health", args)
		keys := make([]string, 0, len(out))
		for k := range out {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		if !reflect.DeepEqual(keys, want) {
			t.Errorf("args %v: keys = %v", args, keys)
		}
		for _, req := range shxSpyRequests(sv) {
			if strings.Contains(req, "sysInfo") || strings.Contains(req, "JMX") || strings.Contains(req, "localHistory") ||
				strings.Contains(req, "securityInfo") || strings.Contains(req, "moduleManager") {
				t.Errorf("args %v: read %s", args, req)
			}
		}
	}
}

// Neither certificate source answers: certificates is null with a note naming both.
func TestStationHealthPlatformCertificatesNullWhenNoSourceAnswers(t *testing.T) {
	r, sv := pfRig(t, licRecorded(t))
	shxServeSpy(sv, pfCryptoPath, boxstation.SpyPage{Status: 403})
	r.Station.Do(func(st *boxstation.Station) {
		services := st.Root().Child("Services")
		services.Children = slices.DeleteFunc(services.Children, func(n *boxstation.Node) bool { return n.Name == "SecurityService" })
	})
	p := pfPlatform(t, r, Args{"platform": true})
	note, _ := p["certificates_note"].(string)
	if v, present := p["certificates"]; !present || v != nil || !strings.Contains(note, "403") || !strings.Contains(note, "SecurityService") {
		t.Errorf("certificates = %v, note %q", v, note)
	}
}

// Live 4.15: the first load of a SystemService not yet polled leaves the CPU usage slots out
// (-1, their default); the load subscribes it, its poll measures them, and a second load reads them.
func TestStationHealthReadsCPUUsageOnceTheServiceIsPolled(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	pfDropCPU(r.Station)
	pfPollCPUOnSecondLoad(r.Station, "58", "62")
	out := r.Srv.OK("n4_station_health", Args{"platform": true})
	system := shxSection(t, out, "system")
	if system["currentCpuUsage"] != 58.0 || system["overallCpuUsage"] != 62.0 || system["cpu_note"] != nil {
		t.Errorf("system = %v", system)
	}
	cpu := shxSection(t, out, "platform")["cpu"]
	if want := map[string]any{"current": 58.0, "overall": 62.0, "cpus": 16.0}; !reflect.DeepEqual(cpu, want) {
		t.Errorf("cpu = %v", cpu)
	}
}

// CPU usage never measured: cpu is null with a cpu_note, never a guessed number.
func TestStationHealthCPUUnmeasuredIsNullWithANote(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	pfDropCPU(r.Station)
	out := r.Srv.OK("n4_station_health", Args{"platform": true})
	if note, _ := shxSection(t, out, "system")["cpu_note"].(string); !strings.Contains(note, "-1") {
		t.Errorf("system cpu_note = %q", note)
	}
	p := shxSection(t, out, "platform")
	note, _ := p["cpu_note"].(string)
	if v, present := p["cpu"]; !present || v != nil || !strings.Contains(note, "poll") || p["memory"] == nil {
		t.Errorf("cpu = %v, note %q, memory %v", v, note, p["memory"])
	}
}
