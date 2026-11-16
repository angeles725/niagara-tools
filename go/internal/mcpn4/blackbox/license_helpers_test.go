package blackbox

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// licPath is the spy page the licence tools read.
const licPath = "sysManagers/licenseManager"

// licNow is the station time of a licence rig: every days_left below is counted from its date.
const licNow = "2026-11-15T19:05:43.877-06:00"

// licRig is a connected servlet rig whose station says it is systemTime ("" for a station with
// no SystemService) and serves page as its licence manager spy page.
func licRig(t *testing.T, systemTime string, page []byte) (*Rig, *boxstation.Servlets) {
	t.Helper()
	return licRigWith(t, systemTime, page, nil)
}

func licRigWith(t *testing.T, systemTime string, page []byte, configure func(o *Options)) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, configure)
	if systemTime != "" {
		shxSystemService(r.Station, map[string][2]string{"systemTime": {"baja:AbsTime", systemTime}})
	}
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: page})
	return r, sv
}

// licRecorded is a licence manager page in the layout measured on a laboratory station (Niagara
// 4.15), cut to a dozen features, every name and value of it neutral.
func licRecorded(t *testing.T) []byte { return shxSpyFixture(t, "license-manager.html") }

// licPage is a licence manager page in the measured layout: a heading, then the lines as
// preformatted text.
func licPage(lines ...string) []byte {
	return []byte("<html><body><h1>Niagara Licensing</h1><pre>\n" + strings.Join(lines, "\n") + "\n</pre></body></html>")
}

// licSmall is a short page with every section: one certificate, licences with the given
// expiries (named A.license, B.license, ...), two features and the brand.
func licSmall(expiries ...string) []byte {
	lines := []string{"HostId=Lab-0000", "", "Certificates", "  Vendor.certificate [expires: never] {valid}", "", "Licenses (perpetual)"}
	for i, e := range expiries {
		lines = append(lines, "  "+string(rune('A'+i))+".license [expires: "+e+"] {valid}")
	}
	return licPage(append(lines, "", "Features", "  tridium:about [expires: never]", "    owner=Lab Owner", "    project=Lab Project",
		"  tridium:brand [expires: never]", "    brandId=LabBrand", "", "Brand", "  brandId = LabBrand")...)
}

// licNamed is the entry of list whose name is name.
func licNamed(t *testing.T, list any, name string) map[string]any {
	t.Helper()
	for _, o := range objects(list) {
		if o["name"] == name {
			return o
		}
	}
	t.Fatalf("no %q in %v", name, list)
	return nil
}
