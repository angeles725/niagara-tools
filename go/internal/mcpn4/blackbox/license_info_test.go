package blackbox

// n4_license_info: what the station's licence manager says, in one call. Every days_left is
// counted from the station's own date.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestLicenseInfoAnswersTheLicencesCertificatesAndIdentityInOneCall(t *testing.T) {
	r, sv := licRig(t, licNow, licRecorded(t))
	out := r.Srv.OK("n4_license_info", nil)
	if out["status"] != "ok" || out["source"] != "spy:/"+licPath || out["host_id"] != "Lab-0000-0000-0000-0000" ||
		out["brand"] != "LabBrand" || out["owner"] != "Lab Owner" || out["project"] != "Lab Project" ||
		out["license_kind"] != "perpetual" || out["station_date"] != "2026-11-15" {
		t.Errorf("out = %v", out)
	}
	if _, noted := out["note"]; noted {
		t.Errorf("a page read whole has a note: %v", out["note"])
	}
	certs := objects(out["certificates"])
	if len(certs) != 3 || !reflect.DeepEqual(certs[2], map[string]any{"name": "Tridium.certificate", "expires": "never", "valid": true}) {
		t.Errorf("certificates = %v", out["certificates"])
	}
	licences := objects(out["licenses"])
	if len(licences) != 3 || !reflect.DeepEqual(licences[2], map[string]any{"name": "Lab.license", "expires": "2027-03-31",
		"valid": true, "days_left": 136.0}) {
		t.Errorf("licenses = %v", out["licenses"])
	}
	wantStrings(t, "spy requests", shxSpyRequests(sv), []string{"/ord?spy:/sysManagers/licenseManager"})
}

func TestLicenseInfoANeverExpiryHasNoDaysLeftAndIsNeverTheNearest(t *testing.T) {
	r, _ := licRig(t, licNow, licSmall("never", "2027-03-31"))
	out := r.Srv.OK("n4_license_info", nil)
	if got := licNamed(t, out["licenses"], "A.license"); got["expires"] != "never" || got["days_left"] != nil {
		t.Errorf("never = %v", got)
	}
	want := map[string]any{"kind": "license", "name": "B.license", "expires": "2027-03-31", "days_left": 136.0, "same_date": 1.0}
	if !reflect.DeepEqual(out["nearest_expiry"], want) || len(objects(out["expired"])) != 0 {
		t.Errorf("nearest = %v, expired = %v", out["nearest_expiry"], out["expired"])
	}
}

func TestLicenseInfoCountsDaysLeftFromTheStationDateAndNamesWhatExpired(t *testing.T) {
	page := licSmall("2026-11-01", "2026-12-15", "2026-11-15")
	r, _ := licRig(t, licNow, page)
	out := r.Srv.OK("n4_license_info", nil)
	for name, days := range map[string]float64{"A.license": -14, "B.license": 30, "C.license": 0} {
		if got := licNamed(t, out["licenses"], name); got["days_left"] != days {
			t.Errorf("%s = %v, want %v days", name, got, days)
		}
	}
	wantExpired := []any{map[string]any{"kind": "license", "name": "A.license", "expires": "2026-11-01", "days_left": -14.0}}
	if !reflect.DeepEqual(out["expired"], wantExpired) || Dig(out, "nearest_expiry", "name") != "A.license" {
		t.Errorf("expired = %v, nearest = %v", out["expired"], out["nearest_expiry"])
	}
	// the same page on a station whose clock says another day: the count follows the station
	later, _ := licRig(t, "2027-01-10T00:30:00.000+02:00", page)
	if got := licNamed(t, later.Srv.OK("n4_license_info", nil)["licenses"], "B.license"); got["days_left"] != -26.0 {
		t.Errorf("on 2027-01-10 B = %v", got)
	}
}

func TestLicenseInfoWithoutAStationDateLeavesDaysLeftUnknown(t *testing.T) {
	r, _ := licRig(t, "", licSmall("2026-11-01"))
	out := r.Srv.OK("n4_license_info", nil)
	note, _ := out["station_date_note"].(string)
	if out["station_date"] != nil || !strings.Contains(note, "systemTime") || out["expired"] != nil ||
		licNamed(t, out["licenses"], "A.license")["days_left"] != nil {
		t.Errorf("out = %v", out)
	}
	if got := out["nearest_expiry"]; Dig(got, "expires") != "2026-11-01" || Dig(got, "days_left") != nil {
		t.Errorf("nearest = %v", got)
	}
}

func TestLicenseInfoCountsALineItCannotReadInTheNote(t *testing.T) {
	r, _ := licRig(t, licNow, licPage("HostId=Lab-0000", "Certificates", "Vendor.certificate [expires: never] {valid}",
		"Licenses", "A.license [expires: 2027-03-31] {valid}", "B.license expires some day", "Features",
		"stray=before any feature", "tridium:about [expires: never]", "owner=Lab Owner", "not an attribute", "Brand", "brandId = LabBrand"))
	out := r.Srv.OK("n4_license_info", nil)
	note, _ := out["note"].(string)
	if !strings.Contains(note, "3 line(s)") || len(objects(out["licenses"])) != 1 || out["owner"] != "Lab Owner" ||
		out["license_kind"] != nil {
		t.Errorf("note = %q, out = %v", note, out)
	}
}

func TestLicenseInfoNamesAMissingSectionInTheNote(t *testing.T) {
	r, sv := licRig(t, licNow, licPage("HostId=Lab-0000", "Licenses (perpetual)", "A.license [expires: 2027-03-31] {valid}"))
	out := r.Srv.OK("n4_license_info", nil)
	note, _ := out["note"].(string)
	if out["status"] != "ok" || !strings.Contains(note, "Certificates, Features, Brand") || len(objects(out["licenses"])) != 1 ||
		len(objects(out["certificates"])) != 0 || out["brand"] != nil {
		t.Errorf("note = %q, out = %v", note, out)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: []byte("<html><body><p>Something else</p></body></html>")})
	out = r.Srv.OK("n4_license_info", nil)
	if _, has := out["licenses"]; out["status"] != "not_recognized" || has || out["hint"] == nil {
		t.Errorf("a page that is no licence page: %v", out)
	}
}

func TestLicenseInfoAnUnreachablePageIsATypedStatusNotAnEmptyLicence(t *testing.T) {
	r, sv := licRig(t, licNow, licRecorded(t))
	for status, page := range map[string]*boxstation.SpyPage{"forbidden": {Status: 403}, "not_found": nil} {
		sv.Do(func(sv *boxstation.Servlets) {
			delete(sv.SpyPages, licPath)
			if page != nil {
				sv.SpyPages[licPath] = *page
			}
		})
		out := r.Srv.OK("n4_license_info", nil)
		if out["status"] != status || out["source"] != "spy:/"+licPath {
			t.Errorf("%s: %v", status, out)
		}
		for _, key := range []string{"host_id", "licenses", "certificates", "features", "nearest_expiry", "expired"} {
			if _, has := out[key]; has {
				t.Errorf("%s: the reply holds %s: %v", status, key, out[key])
			}
		}
	}
}

func TestLicenseInfoCapsAndFiltersTheFeatureList(t *testing.T) {
	r, _ := licRig(t, licNow, licRecorded(t))
	features := shxSection(t, r.Srv.OK("n4_license_info", Args{"max_features": 3}), "features")
	if features["count"] != 12.0 || features["matched"] != 12.0 || len(objects(features["list"])) != 3 || features["truncated"] != true {
		t.Errorf("capped = %v", features)
	}
	want := []any{map[string]any{"name": "tridium:jsontoolkit", "expires": "2027-03-31", "attributes": map[string]any{"import": "true", "export": "true", "sma.exempt": "true"}}}
	features = shxSection(t, r.Srv.OK("n4_license_info", Args{"feature": "tridium:jsontoolkit"}), "features")
	if features["count"] != 12.0 || features["matched"] != 1.0 || features["truncated"] != false || !reflect.DeepEqual(features["list"], want) {
		t.Errorf("exact = %v", features)
	}
	features = shxSection(t, r.Srv.OK("n4_license_info", Args{"feature": "BACnet"}), "features")
	names := sortedStrings(features["list"], "name")
	if !reflect.DeepEqual(names, []string{"tridium:bacnet", "vendora:bacnetutil"}) || features["matched"] != float64(len(names)) || features["truncated"] != false {
		t.Errorf("substring = %v", features)
	}
	for _, name := range names {
		if !strings.Contains(name, "bacnet") {
			t.Errorf("%q does not match bacnet", name)
		}
	}
	if features = shxSection(t, r.Srv.OK("n4_license_info", Args{"max_features": 0}), "features"); len(objects(features["list"])) != 0 ||
		features["truncated"] != true {
		t.Errorf("max_features=0 = %v", features)
	}
}

func TestLicenseInfoReportsTheMaintenanceFactsTheLicenceStates(t *testing.T) {
	r, sv := licRig(t, licNow, licRecorded(t))
	got := shxSection(t, r.Srv.OK("n4_license_info", nil), "maintenance")
	exempt := map[string]any{"sma.exempt": "true"}
	wantAttrs := []any{map[string]any{"feature": "tridium:cloudlink", "attributes": exempt},
		map[string]any{"feature": "tridium:http", "attributes": exempt}, map[string]any{"feature": "tridium:jsontoolkit", "attributes": exempt}}
	_, boolean := got["exempt"]
	wantExempt := []any{"tridium:cloudlink", "tridium:http", "tridium:jsontoolkit"}
	if boolean || got["sma_feature_count"] != 3.0 || got["sma_exempt_count"] != 3.0 || got["sma_truncated"] != false ||
		!reflect.DeepEqual(got["sma_exempt_features"], wantExempt) || !reflect.DeepEqual(got["sma_attributes"], wantAttrs) || got["feature"] != nil || got["expires"] != nil ||
		got["note"] == nil {
		t.Errorf("maintenance = %v", got)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licPage("HostId=Lab-0000", "Features", "tridium:maintenance [expires: 2026-12-15]",
		"level=full", "tridium:http [expires: never]", "sma.exempt=false", "tridium:mqtt [expires: never]", "sma.exempt=true")})
	got = shxSection(t, r.Srv.OK("n4_license_info", nil), "maintenance")
	wantFeature := map[string]any{"name": "tridium:maintenance", "expires": "2026-12-15", "days_left": 30.0,
		"attributes": map[string]any{"level": "full"}}
	if !reflect.DeepEqual(got["sma_exempt_features"], []any{"tridium:mqtt"}) || len(objects(got["sma_attributes"])) != 2 ||
		!reflect.DeepEqual(got["feature"], wantFeature) {
		t.Errorf("maintenance = %v", got)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licSmall("never")})
	if got = shxSection(t, r.Srv.OK("n4_license_info", nil), "maintenance"); len(got["sma_exempt_features"].([]any)) != 0 || len(objects(got["sma_attributes"])) != 0 {
		t.Errorf("no sma attribute: %v", got)
	}
}

func TestLicenseInfoSaysAPageThatWasCutMayMissFeatures(t *testing.T) {
	lines := []string{"HostId=Lab-0000", "Features"}
	for i := 0; i < 50000; i++ {
		lines = append(lines, "tridium:feature"+strings.Repeat("x", 20)+" [expires: never]")
	}
	r, _ := licRig(t, licNow, licPage(lines...))
	out := r.Srv.OK("n4_license_info", Args{"max_features": 1})
	note, _ := out["note"].(string)
	if out["truncated"] != true || !strings.Contains(note, "cut") {
		t.Errorf("truncated = %v, note = %q", out["truncated"], note)
	}
}

func TestLicenseInfoIsAReadToolThatNeedsASession(t *testing.T) {
	r := NewRig(t, nil)
	if tool := toolsByName(t, r.Srv)["n4_license_info"]; Dig(tool, "annotations", "readOnlyHint") != true {
		t.Errorf("tools/list entry = %v", tool)
	}
	if text := r.Srv.Err("n4_license_info", nil); !strings.Contains(text, "n4_connect") {
		t.Errorf("without a session: %q", text)
	}
}

func TestLicenseInfoWithARefusedLoginIsAnAuthError(t *testing.T) {
	r, sv := licRig(t, licNow, licRecorded(t))
	sv.SetPassword("changed")
	if text := r.Srv.Err("n4_license_info", nil); !strings.Contains(text, "401") {
		t.Errorf("error = %q", text)
	}
}
