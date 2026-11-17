package blackbox

// n4_station_health platform=true when a source fails or the station's date is unknown.

import (
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// A transport or HTTP failure of the module manager page (here a 500) leaves modules null with a
// note, like every other platform item; it never fails the call.
func TestStationHealthPlatformModulesPageFailureIsNullWithANote(t *testing.T) {
	r, sv := pfRig(t, licRecorded(t))
	shxServeSpy(sv, "sysManagers/moduleManager", boxstation.SpyPage{Body: []byte("boom"), Status: 500})
	res := r.Srv.Call("n4_station_health", Args{"platform": true})
	if res.IsError {
		t.Fatalf("isError: %s", res.Text)
	}
	p := pfPlatform(t, r, Args{"platform": true})
	note, _ := p["modules_note"].(string)
	if v, present := p["modules"]; !present || v != nil || !strings.Contains(note, "500") {
		t.Errorf("modules = %v, note %q", v, note)
	}
}

// A station certificate whose expiry point is down has expired, even on its last day (0 days).
func TestStationHealthPlatformDownStationCertificateIsExpired(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	pfStationCerts(r.Station, map[string]string{"lastday": "0,0 {down}"})
	p := pfPlatform(t, r, Args{"platform": true})
	if !pfWarned(p, "lastday", "expired") || pfWarned(p, "lastday", "in 0 day") {
		t.Errorf("warnings = %v", p["warnings"])
	}
}

// With the station's date unknown a station certificate has no date: its warning says the days
// alone, and it is not counted as an unreadable Expires cell of the Cryptography Info page.
func TestStationHealthPlatformUnknownClockWarnsByDaysAlone(t *testing.T) {
	r, _ := pfRig(t, licRecorded(t))
	r.Station.Do(func(st *boxstation.Station) {
		svc := st.Root().Child("Services").Child("PlatformServices").Child("SystemService")
		svc.Children = slices.DeleteFunc(svc.Children, func(n *boxstation.Node) bool { return n.Name == "systemTime" })
	})
	p := pfPlatform(t, r, Args{"platform": true})
	for _, w := range p["warnings"].([]any) {
		if s := w.(string); strings.Contains(s, "expires ,") || strings.Contains(s, "expired ,") {
			t.Errorf("warning with an empty date: %q", s)
		}
	}
	if !pfWarned(p, "tlsrenew", "10 day") {
		t.Errorf("warnings = %v", p["warnings"])
	}
	if note := p["certificates_note"].(string); strings.Contains(note, "Expires cell") {
		t.Errorf("certificates_note = %q", note)
	}
}
