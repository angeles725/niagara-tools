package blackbox

// The license section of n4_station_health: the licences with their days left, the nearest
// expiry, what expired and a warning inside the window, answered on its own like every section.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// licHealthRig is the health rig whose station also serves page as its licence manager page.
func licHealthRig(t *testing.T, page []byte) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := shxHealthRig(t)
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: page})
	return r, sv
}

func TestStationHealthCarriesTheLicencesWithTheirDaysLeft(t *testing.T) {
	r, _ := licHealthRig(t, licRecorded(t))
	want := map[string]any{
		"licenses": []any{map[string]any{"name": "VendorA.license", "expires": "2027-03-31", "valid": true, "days_left": 136.0},
			map[string]any{"name": "VendorB.license", "expires": "2027-03-31", "valid": true, "days_left": 136.0},
			map[string]any{"name": "Lab.license", "expires": "2027-03-31", "valid": true, "days_left": 136.0}},
		"nearest_expiry": map[string]any{"kind": "license", "name": "VendorA.license", "expires": "2027-03-31",
			"days_left": 136.0, "same_date": 15.0},
		"expired": []any{}, "source": "spy:/" + licPath}
	if got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license"); !reflect.DeepEqual(got, want) {
		t.Errorf("license = %v", got)
	}
}

func TestStationHealthWarnsWhenTheNearestExpiryIsWithinSixtyDays(t *testing.T) {
	r, sv := licHealthRig(t, licSmall("2027-01-15", "never")) // 61 days from the station date
	if got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license"); got["warning"] != nil {
		t.Errorf("61 days away: %v", got)
	}
	for expires, say := range map[string]string{"2027-01-14": "60 day(s)", "2026-12-15": "30 day(s)", "2026-11-15": "0 day(s)",
		"2026-11-01": "expired 14 day(s) ago"} {
		shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("2027-03-31", expires)})
		got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")
		if warning, _ := got["warning"].(string); !strings.Contains(warning, say) || !strings.Contains(warning, "B.license") {
			t.Errorf("%s: warning = %q", expires, warning)
		}
		if expired := objects(got["expired"]); (len(expired) == 1) != (expires == "2026-11-01") {
			t.Errorf("%s: expired = %v", expires, got["expired"])
		}
	}
}

func TestStationHealthReportsAFailingLicenceSectionAndStillAnswersTheOthers(t *testing.T) {
	r, sv := licHealthRig(t, licRecorded(t))
	for name, page := range map[string]boxstation.SpyPage{"forbidden": {Status: 403},
		"not_recognized": {Body: []byte("<html><body><p>Something else</p></body></html>")}} {
		shxServeSpy(sv, licPath, page)
		out := r.Srv.OK("n4_station_health", nil)
		license := shxSection(t, out, "license")
		if text, _ := license["error"].(string); len(license) != 1 || !strings.Contains(text, name) {
			t.Errorf("%s: license = %v", name, license)
		}
		if shxSection(t, out, "points")["total"] != 5.0 || shxSection(t, out, "system")["stationName"] != "LabStation" ||
			len(objects(shxSection(t, out, "capacity")["resources"])) != 6 {
			t.Errorf("%s: the other sections did not answer: %v", name, out)
		}
	}
}

func TestStationHealthLicenceWithoutAStationDateHasNoDaysLeftAndSaysWhy(t *testing.T) {
	r, sv := servletRig(t, nil)
	shxRouteHealth(sv, shxNetworksCSV, shxDevicesCSV, shxPointsCSV)
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("2026-11-01")})
	got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")
	note, _ := got["note"].(string)
	if !strings.Contains(note, "systemTime") || got["expired"] != nil || got["warning"] != nil ||
		Dig(got, "nearest_expiry", "days_left") != nil || Dig(got, "nearest_expiry", "expires") != "2026-11-01" {
		t.Errorf("license = %v", got)
	}
}

func TestStationHealthLicenceNotesWhatThePageReadingCouldNotDo(t *testing.T) {
	r, _ := licHealthRig(t, licPage("HostId=Lab-0000", "Licenses", "A.license [expires: 2027-03-31] {valid}", "what is this"))
	got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")
	if note, _ := got["note"].(string); !strings.Contains(note, "1 line(s)") || !strings.Contains(note, "Certificates, Features, Brand") {
		t.Errorf("note = %q", note)
	}
}
