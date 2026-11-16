package blackbox

// What is not a plain valid, dated licence is said, in n4_license_info and in the health section:
// a state other than valid, an expiry that is not a date, a station time that is not a time.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// licStates is a page whose licences are in states other than valid, with one valid.
func licStates() []byte {
	return licPage("HostId=H", "Certificates", "Old.certificate [expires: 2020-01-01] {expired}", "Licenses",
		"A.license [expires: 2020-01-01] {expired}", "B.license [expires: never] {invalid}", "C.license [expires: 2027-03-31] {valid}",
		"D.license [expires: never] {Valid}", "E.license [expires: never] {}")
}

func TestLicenseInfoReportsALicenceThatIsNotValidAsSuch(t *testing.T) {
	r, _ := licRig(t, licNow, licStates())
	out := r.Srv.OK("n4_license_info", nil)
	for name, state := range map[string]string{"A.license": "expired", "B.license": "invalid", "D.license": "Valid", "E.license": ""} {
		if got := licNamed(t, out["licenses"], name); got["valid"] != false || got["state"] != state {
			t.Errorf("%s = %v", name, got)
		}
	}
	want := map[string]any{"name": "C.license", "expires": "2027-03-31", "valid": true, "days_left": 136.0}
	if got := licNamed(t, out["licenses"], "C.license"); !reflect.DeepEqual(got, want) {
		t.Errorf("C = %v", got)
	}
	wantCert := map[string]any{"name": "Old.certificate", "expires": "2020-01-01", "valid": false, "state": "expired"}
	if got := licNamed(t, out["certificates"], "Old.certificate"); !reflect.DeepEqual(got, wantCert) {
		t.Errorf("certificate = %v", got)
	}
}

func TestStationHealthSaysWhichLicencesAreNotValid(t *testing.T) {
	r, sv := licHealthRig(t, licStates())
	got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")
	warning, _ := got["warning"].(string)
	if a := licNamed(t, got["licenses"], "A.license"); a["valid"] != false || a["state"] != "expired" ||
		licNamed(t, got["licenses"], "C.license")["valid"] != true || !strings.Contains(warning, "not valid: A.license (expired), B.license (invalid)") {
		t.Errorf("license = %v", got)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("2027-03-31")})
	if got = shxSection(t, r.Srv.OK("n4_station_health", nil), "license"); got["warning"] != nil {
		t.Errorf("every licence valid and far from expiry: warning = %v", got["warning"])
	}
}

func TestStationHealthFailsWhenTheLicencePageRefusesTheLogin(t *testing.T) {
	r, sv := licHealthRig(t, licSmall("2027-03-31"))
	shxServeSpy(sv, licPath, boxstation.SpyPage{Status: 401})
	if text := r.Srv.Err("n4_station_health", nil); !strings.Contains(text, "401") {
		t.Errorf("error = %q", text)
	}
}

func TestLicenseInfoReportsAnExpiryItCannotReadAsADate(t *testing.T) {
	r, sv := licRig(t, licNow, licPage("HostId=H", "Licenses", "A.license [expires: tomorrow] {valid}", "B.license [expires: 2027-02-30] {valid}",
		"C.license [expires: Never] {valid}", "D.license [expires: 2027-03-31] {valid}", "Features", "tridium:x [expires: 31/03/2027]"))
	out := r.Srv.OK("n4_license_info", nil)
	for name, raw := range map[string]string{"A.license": "tomorrow", "B.license": "2027-02-30", "C.license": "Never"} {
		if got := licNamed(t, out["licenses"], name); got["expires"] != raw || got["days_left"] != nil {
			t.Errorf("%s = %v", name, got)
		}
	}
	if note, _ := out["note"].(string); !strings.Contains(note, "4 expiry value(s)") || Dig(out, "nearest_expiry", "name") != "D.license" {
		t.Errorf("note = %q, nearest = %v", note, out["nearest_expiry"])
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("9999-12-31", "0001-01-01")})
	out = r.Srv.OK("n4_license_info", nil)
	if a, b := licNamed(t, out["licenses"], "A.license"), licNamed(t, out["licenses"], "B.license"); a["days_left"] != 2912124.0 ||
		b["days_left"] != -739934.0 {
		t.Errorf("far dates: %v, %v", a, b)
	}
}

func TestStationHealthWarnsOfAnExpiryItCouldNotRead(t *testing.T) {
	r, _ := licHealthRig(t, licPage("HostId=H", "Licenses", "A.license [expires: tomorrow] {valid}", "B.license [expires: 2027-03-31] {valid}"))
	got := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")
	if warning, _ := got["warning"].(string); !strings.Contains(warning, "1 expiry value(s)") || licNamed(t, got["licenses"], "A.license")["days_left"] != nil {
		t.Errorf("license = %v", got)
	}
}

func TestLicenseInfoNeedsAWholeStationTimeForDaysLeft(t *testing.T) {
	for _, bad := range []string{"2026-11-15garbage", "2026-11-15", "garbage", "2026-02-30T00:00:00.000+00:00", "2026-11-15T19:05:43.877-06:00x"} {
		r, _ := licRig(t, bad, licSmall("2026-11-01"))
		out := r.Srv.OK("n4_license_info", nil)
		if note, _ := out["station_date_note"].(string); out["station_date"] != nil || !strings.Contains(note, "systemTime") ||
			licNamed(t, out["licenses"], "A.license")["days_left"] != nil || out["expired"] != nil {
			t.Errorf("%q: station_date %v, note %q, licences %v", bad, out["station_date"], note, out["licenses"])
		}
	}
	r, _ := licRig(t, "2026-11-15T23:30:00.000+14:00", licSmall("2026-11-16"))
	if out := r.Srv.OK("n4_license_info", nil); out["station_date"] != "2026-11-15" || licNamed(t, out["licenses"], "A.license")["days_left"] != 1.0 {
		t.Errorf("the station's own day: %v", out)
	}
}

func TestStationHealthLicenceErrorSaysWhatHappenedForEachStatus(t *testing.T) {
	r, sv := licHealthRig(t, licSmall("2027-03-31"))
	plain := boxstation.SpyPage{Body: []byte("HostId=H"), ContentType: "text/plain"}
	other := boxstation.SpyPage{Body: []byte("<html><body><p>Something else</p></body></html>")}
	for status, page := range map[string]*boxstation.SpyPage{"not_recognized": &other, "not_html": &plain, "not_found": nil, "forbidden": {Status: 403}} {
		sv.Do(func(sv *boxstation.Servlets) {
			delete(sv.SpyPages, licPath)
			if page != nil {
				sv.SpyPages[licPath] = *page
			}
		})
		text, _ := shxSection(t, r.Srv.OK("n4_station_health", nil), "license")["error"].(string)
		if blames := strings.Contains(text, "allowed to read spy pages"); !strings.Contains(text, status) || blames != (status == "forbidden") {
			t.Errorf("%s: %q", status, text)
		}
	}
}
