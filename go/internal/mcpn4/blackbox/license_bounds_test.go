package blackbox

// n4_license_info and the health licence section stay small whatever the page holds, and never
// read half a line.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// licReplyBound is the stated bound of a whole licence reply (the JSON-RPC result), in bytes,
// for any page up to the byte cap of a spy read.
const licReplyBound = 2 << 20

// licReplySize is the size of a call's whole result as the server writes it: JSON with every
// character outside ASCII as \uXXXX escapes (two for a character outside the BMP).
func licReplySize(t *testing.T, res Result) int {
	t.Helper()
	raw, err := json.Marshal(res.Raw)
	if err != nil {
		t.Fatal(err)
	}
	size := 0
	for _, r := range string(raw) {
		switch {
		case r < 0x80:
			size++
		case r < 0x10000:
			size += 6
		default:
			size += 12
		}
	}
	return size
}

// licAmplifier is a page of one feature with a name of nameLen characters and attrs sma.a= lines.
func licAmplifier(nameLen, attrs int) []byte {
	lines := []string{"HostId=H", "Features", "v:" + strings.Repeat("n", nameLen) + " [expires: never]"}
	for i := 0; i < attrs; i++ {
		lines = append(lines, "sma.a=")
	}
	return licPage(lines...)
}

func TestLicenseInfoReplyIsNotQuadraticInThePage(t *testing.T) {
	r, sv := licRig(t, licNow, licSmall("never"))
	for _, c := range [][2]int{{2000, 1000}, {2000, 5000}, {20000, 1500}, {2000, 100000}} {
		body := licAmplifier(c[0], c[1])
		shxServeSpy(sv, licPath, boxstation.SpyPage{Body: body})
		res := r.Srv.Call("n4_license_info", Args{"max_features": 0})
		if size := licReplySize(t, res); res.IsError || size > 16<<10 || size > licReplyBound {
			t.Errorf("name %d, %d attributes: a page of %d bytes gave a reply of %d bytes", c[0], c[1], len(body), size)
		}
		m := shxSection(t, res.Structured, "maintenance")
		groups := objects(m["sma_attributes"])
		if len(groups) != 1 || !reflect.DeepEqual(groups[0]["attributes"], map[string]any{"sma.a": ""}) || m["sma_feature_count"] != 1.0 {
			t.Errorf("sma_attributes = %v (count %v)", m["sma_attributes"], m["sma_feature_count"])
		}
		if name, _ := groups[0]["feature"].(string); len(name) > 220 || !strings.HasSuffix(name, "...[cut]") {
			t.Errorf("the feature name was not cut with a marker: %d characters", len(name))
		}
	}
}

// Pages built to make every list and every string as costly as it can be (names of quotes and
// of characters outside the BMP, which the reply escapes) stay under the stated bound.
func TestLicenseInfoWholeReplyStaysUnderTheStatedBound(t *testing.T) {
	r, sv := licRig(t, licNow, licSmall("never"))
	costly := func(n int) string { return strings.Repeat("\U0001F512\"\\", 80) + fmt.Sprint(n) }
	pages := map[string][]string{}
	many := []string{"HostId=" + costly(0), "Certificates"}
	for i := 0; i < 900; i++ {
		many = append(many, costly(i)+".certificate [expires: 2020-01-01] {"+costly(i)+"}")
	}
	many = append(many, "Licenses ("+costly(1)+")")
	for i := 0; i < 900; i++ {
		many = append(many, costly(i)+".license [expires: 2020-01-01] {"+costly(i)+"}")
	}
	pages["entries"] = many
	features := []string{"HostId=H", "Features"}
	for i := 0; i < 350; i++ {
		features = append(features, "v:"+costly(i)+" [expires: 2020-01-01]")
		for a := 0; a < 5; a++ {
			features = append(features, fmt.Sprintf("sma.%s%d=%s", costly(a), a, costly(a)), "sma.exempt=true")
		}
	}
	pages["features"] = features
	licences := []string{"HostId=H", "Licenses"}
	for i := 0; i < 40000; i++ {
		licences = append(licences, fmt.Sprintf("L%d.license [expires: 2020-01-01] {valid}", i))
	}
	pages["licences"] = licences
	for name, lines := range pages {
		body := licPage(lines...)
		if len(body) > 2<<20 {
			t.Fatalf("%s: the page is %d bytes, over the byte cap", name, len(body))
		}
		shxServeSpy(sv, licPath, boxstation.SpyPage{Body: body})
		res := r.Srv.Call("n4_license_info", Args{"max_features": 2000})
		if size := licReplySize(t, res); res.IsError || size > licReplyBound {
			t.Errorf("%s: a page of %d bytes gave a reply of %d bytes (error %v)", name, len(body), size, res.IsError)
		} else {
			t.Logf("%s: a page of %d bytes gave a reply of %d bytes", name, len(body), size)
		}
		if size := licReplySize(t, r.Srv.Call("n4_station_health", nil)); size > licReplyBound {
			t.Errorf("%s: health reply of %d bytes", name, size)
		}
	}
	out := r.Srv.OK("n4_license_info", nil) // the 40 000 licences
	if got := len(objects(out["licenses"])); got > 100 || out["licenses_count"] != 40000.0 || out["expired_count"] != 100.0 {
		t.Errorf("%d licences listed, licenses_count %v, expired_count %v", got, out["licenses_count"], out["expired_count"])
	}
	if note, _ := out["note"].(string); !strings.Contains(note, "40000") {
		t.Errorf("note = %q", note)
	}
}

func TestLicenseInfoCapsTheMaintenanceListsAndNamesEachExemptFeatureOnce(t *testing.T) {
	lines := []string{"HostId=H", "Features"}
	for i := 0; i < 120; i++ {
		lines = append(lines, fmt.Sprintf("v:f%03d [expires: never]", i), "sma.exempt=true", "sma.exempt=true", "sma.level=1")
	}
	lines = append(lines, "v:f000 [expires: never]", "sma.exempt=true") // the same name again
	r, _ := licRig(t, licNow, licPage(lines...))
	m := shxSection(t, r.Srv.OK("n4_license_info", Args{"max_features": 0}), "maintenance")
	exempt, _ := m["sma_exempt_features"].([]any)
	if len(exempt) != 50 || exempt[0] != "v:f000" || exempt[1] != "v:f001" || m["sma_exempt_count"] != 120.0 ||
		len(objects(m["sma_attributes"])) != 50 || m["sma_feature_count"] != 121.0 || m["sma_truncated"] != true {
		t.Errorf("exempt %d of %v, attributes %d of %v, truncated %v", len(exempt), m["sma_exempt_count"],
			len(objects(m["sma_attributes"])), m["sma_feature_count"], m["sma_truncated"])
	}
	if got := objects(m["sma_attributes"])[0]; !reflect.DeepEqual(got, map[string]any{"feature": "v:f000",
		"attributes": map[string]any{"sma.exempt": "true", "sma.level": "1"}}) {
		t.Errorf("first group = %v", got)
	}
}

func TestLicenseInfoCutsALongAttributeListAndSaysSo(t *testing.T) {
	lines := []string{"HostId=H", "Features", "v:wide [expires: never]"}
	for i := 0; i < 45; i++ {
		lines = append(lines, fmt.Sprintf("k%02d=%s", i, strings.Repeat("v", 150)))
	}
	r, _ := licRig(t, licNow, licPage(lines...))
	entry := objects(shxSection(t, r.Srv.OK("n4_license_info", nil), "features")["list"])[0]
	attrs, _ := entry["attributes"].(map[string]any)
	if value, _ := attrs["k00"].(string); len(attrs) != 30 || entry["attributes_truncated"] != true || !strings.HasSuffix(value, "...[cut]") || len(value) != 108 {
		t.Errorf("%d attributes, truncated %v, k00 of %d characters", len(attrs), entry["attributes_truncated"], len(attrs["k00"].(string)))
	}
}

// A line is read whole or not at all: two entries on one line, a name with a bracket or a
// brace, a state after a feature are counted as unread, never turned into a valid entry.
func TestLicenseInfoNeverReadsHalfALine(t *testing.T) {
	r, sv := licRig(t, licNow, licPage("HostId=H", "Certificates",
		"A.certificate [expires: 2020-01-01] {invalid} B.certificate [expires: never] {valid}", "Licenses",
		"A.license [expires: 2020-01-01] {invalid} B.license [expires: never] {valid}", "C]x.license [expires: never] {valid}",
		"D.license [expires: never] {va{lid}", "E.license [expires: ne]ver] {valid}", "Features",
		"tridium:a [expires: 2020-01-01] tridium:b [expires: never]", "tridium:c [expires: never] {valid}", "tridium:d[x] [expires: never]"))
	out := r.Srv.OK("n4_license_info", nil)
	note, _ := out["note"].(string)
	if len(objects(out["licenses"])) != 0 || len(objects(out["certificates"])) != 0 || Dig(out, "features", "count") != 0.0 ||
		!strings.Contains(note, "8 line(s)") {
		t.Errorf("licenses %v, certificates %v, features %v, note %q", out["licenses"], out["certificates"], Dig(out, "features", "count"), note)
	}
	// the same two entries as markup with no line break between them
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: []byte("<html><body><h1>Niagara Licensing</h1><div>HostId=H</div><div>Licenses</div>" +
		"<span>A.license [expires: 2020-01-01] {invalid}</span> <span>B.license [expires: never] {valid}</span></body></html>")})
	out = r.Srv.OK("n4_license_info", nil)
	if note, _ = out["note"].(string); len(objects(out["licenses"])) != 0 || !strings.Contains(note, "1 line(s)") {
		t.Errorf("licenses %v, note %q", out["licenses"], note)
	}
}

func TestLicenseInfoASectionHeadingOnlyCountsWhereTheLayoutAllowsIt(t *testing.T) {
	r, sv := licRig(t, licNow, licPage("HostId=One", "HostId=Two", "Licenses (trial) (other)", "A.license [expires: 2020-01-01] {invalid}",
		"Features", "tridium:x [expires: never]", "note=1", "Licenses", "Z.license [expires: never] {valid}", "Certificates",
		"C.certificate [expires: never] {valid}", "HostId=Late"))
	out := r.Srv.OK("n4_license_info", nil)
	note, _ := out["note"].(string)
	if out["host_id"] != "One" || out["license_kind"] != "trial" || len(objects(out["licenses"])) != 1 ||
		licNamed(t, out["licenses"], "A.license")["valid"] != false || len(objects(out["certificates"])) != 0 ||
		!strings.Contains(note, "5 line(s)") || !strings.Contains(note, "Certificates, Brand") {
		t.Errorf("out = %v", out)
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licPage("Brand", "brandId = X", "Licenses", "A.license [expires: never] {valid}")})
	if out = r.Srv.OK("n4_license_info", nil); len(objects(out["licenses"])) != 0 || out["brand"] != "X" {
		t.Errorf("a Licences heading after the Brand section was read: %v", out)
	}
}

func TestLicenseInfoStripsControlCharactersFromWhatItReturns(t *testing.T) {
	r, _ := licRig(t, licNow, licPage("HostId=H\x1b[31m\x00", "Licenses (per\x07petual)", "A\x07.license [expires: 2027-03-31\x00] {valid}",
		"Features", "tridium:about [expires: never]", "owner=Lab\x1b Owner\u0085\u009b"))
	res := r.Srv.Call("n4_license_info", nil)
	for _, r := range res.Text {
		if r < 0x20 || r >= 0x7f && r < 0xa0 {
			t.Fatalf("the reply holds the control character %U", r)
		}
	}
	out := res.Structured
	if out["host_id"] != "H[31m" || out["license_kind"] != "perpetual" || out["owner"] != "Lab Owner" ||
		licNamed(t, out["licenses"], "A.license")["days_left"] != 136.0 {
		t.Errorf("out = %v", out)
	}
}
