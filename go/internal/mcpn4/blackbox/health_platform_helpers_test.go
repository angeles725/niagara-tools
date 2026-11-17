package blackbox

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// The spy pages n4_station_health platform=true reads, in the layouts of their 4.15 sources
// (SysInfoSpy, UtilSpy.JmxInfo, BLocalHistoryDatabase.SpyLocalHistoryDatabase,
// CryptoPlatformPage), every name neutral. The station's day (shxHealthRig) is 2026-11-15.
const (
	pfJmxPath     = "util/JMX Info"
	pfHistoryPath = "localHistory"
	pfCryptoPath  = "securityInfo/Cryptography Info"
)

// pfJmxPage is a JMX Info page with a G1 heap of maximum oldMax KB (eden and survivor maxima
// undefined, printed 0) and two non-heap pools.
func pfJmxPage(oldMax string) []byte {
	pool := func(name, max string) string {
		return "<table><tr><th colspan='5'>" + name + " (KB)</th></tr><tr><th>Type</th><th>init</th><th>used</th>" +
			"<th>committed</th><th>max</th></tr><tr><td>current</td><td>1</td><td>2</td><td>3</td><td>" + max +
			"</td></tr><tr><td>peak</td><td>1</td><td>2</td><td>3</td><td>" + max + "</td></tr></table>"
	}
	return []byte("<html><body><table><tr><th colspan='2'>JMX Thread Info</th></tr><tr><td>peak thread count</td><td>90</td></tr></table>" +
		pool("G1 Eden Space", "0") + pool("G1 Survivor Space", "0") + pool("G1 Old Gen", oldMax) +
		pool("Metaspace", "0") + pool("CodeHeap 'non-nmethods'", "5836") + "</body></html>")
}

// pfHistoryPage is a localHistory page with two histories of 100 and 250 records.
func pfHistoryPage() []byte {
	return []byte("<html><body><table><tr><th colspan='2'>LocalHistoryDatabase</th></tr><tr><td>dbDir</td><td>file:^history</td></tr>" +
		"<tr><td>closing</td><td>false</td></tr></table>NOTE: Statistics are collected only since station boot AND while history is open" +
		"<table><tr><th>Name</th><th>Records</th><th>Total Appends</th><th>Header Writes</th><th>Page Writes</th></tr>" +
		"<tr><td><a href='/ord?spy:/localHistory/AhuTemp'>AhuTemp</a></td><td>100</td><td>4</td><td>1</td><td>2</td></tr>" +
		"<tr><td><a href='/ord?spy:/localHistory/AhuFan'>AhuFan</a></td><td>250</td><td>0</td><td>0</td><td>0</td></tr>" +
		"</table></body></html>")
}

// pfCryptoPage is a Cryptography Info page: one system CA valid to 2035, three expired JDK
// system roots, and two user
// certificates, one expiring 2026-12-01 (16 days) and one expired 2026-01-05 (314 days ago).
func pfCryptoPage() []byte {
	head := func(title string) string {
		return "<table><tr><th colspan='4'>" + title + "</th></tr><tr><th>Alias</th><th>Issued To</th><th>Issued By</th><th>Expires</th></tr>"
	}
	row := func(alias, issuer, expires string) string {
		return "<tr><td><a href='/ord?spy:/securityInfo/Cryptography%20Info/" + alias + "'>" + alias + "</a></td><td>" + alias +
			"</td><td>" + issuer + "</td><td>" + expires + "</td></tr>"
	}
	return []byte("<html><body><table><tr><th colspan='2'>FIPS</th></tr><tr><td>FIPS enabled</td><td>false</td></tr></table>" +
		head("Trusted System Certificates (4 certs)") + row("labrootca", "CN=Lab Root CA", "Sat Mar 31 17:59:59 CST 2035") +
		row("oldroot1 [jdk]", "CN=Old Root 1", "Sat May 30 04:48:38 CST 2020") +
		row("oldroot2 [jdk]", "CN=Old Root 2", "Tue Jul 09 12:19:22 CDT 2019") +
		row("oldroot3 [jdk]", "CN=Old Root 3", "Mon Jan 05 10:00:00 CST 2026") + "</table>" +
		head("Trusted User Certificates (2 certs)") + row("labserverca", "CN=Lab Server CA", "Tue Dec 01 10:00:00 CST 2026") +
		row("oldvendor", "CN=Old Vendor", "Mon Jan 05 10:00:00 CST 2026") + "</table></body></html>")
}

// pfRig is the health rig with every page platform=true reads, the licence page given.
func pfRig(t *testing.T, licence []byte) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := shxHealthRig(t)
	shxServeSpy(sv, "sysInfo", boxstation.SpyPage{Body: shxSpyFixture(t, "sysinfo.html")})
	shxServeSpy(sv, pfJmxPath, boxstation.SpyPage{Body: pfJmxPage("2097152")})
	shxServeSpy(sv, pfHistoryPath, boxstation.SpyPage{Body: pfHistoryPage()})
	shxServeSpy(sv, pfCryptoPath, boxstation.SpyPage{Body: pfCryptoPage()})
	shxServeSpy(sv, "sysManagers/moduleManager", boxstation.SpyPage{Body: shxSpyFixture(t, "module-manager.html")})
	shxServeSpy(sv, licPath, boxstation.SpyPage{Body: licence})
	pfStationCerts(r.Station, map[string]string{"default": "329,0 {ok}", "tlsrenew": "10,0 {ok}"})
	return r, sv
}

// pfStationCerts gives the station a Services/SecurityService/certificates folder with one
// certificate info per alias, its expiry point holding the given value text (days left, as a
// 4.15 Spanish-locale station writes it).
func pfStationCerts(st *boxstation.Station, expiries map[string]string) {
	st.Do(func(st *boxstation.Station) {
		node := st.Root()
		for _, name := range []string{"Services", "SecurityService", "certificates"} {
			if child := node.Child(name); child != nil {
				node = child
			} else {
				node = st.Component(node, name, "baja:Component", "")
			}
		}
		for alias, value := range expiries {
			info := st.Component(node, alias, "nss:CertificateInfo", "")
			info.Children = append(info.Children, &boxstation.Node{Name: "alias", Type: "baja:String", Value: alias},
				&boxstation.Node{Name: "expiry", Type: "nss:CertificateExpiryPoint", Value: value})
		}
	})
}

// pfDropCPU removes the CPU usage slots of the SystemService, as a load of a service still at
// -1 (not polled yet) leaves them out.
func pfDropCPU(st *boxstation.Station) {
	st.Do(func(st *boxstation.Station) {
		svc := st.Root().Child("Services").Child("PlatformServices").Child("SystemService")
		kept := svc.Children[:0]
		for _, c := range svc.Children {
			if c.Name != "currentCpuUsage" && c.Name != "overallCpuUsage" {
				kept = append(kept, c)
			}
		}
		svc.Children = kept
	})
}

// pfPollCPUOnSecondLoad models the service's poll on subscribe: the second load of the
// SystemService finds the CPU usage slots measured (current, overall).
func pfPollCPUOnSecondLoad(st *boxstation.Station, current, overall string) {
	loads := 0
	st.SetHook(func(frame map[string]any) (int, []byte) {
		raw, _ := json.Marshal(frame)
		if strings.Contains(string(raw), `"loadSlots"`) && strings.Contains(string(raw), "PlatformServices/SystemService") {
			loads++
			if loads == 2 {
				st.Do(func(st *boxstation.Station) {
					svc := st.Root().Child("Services").Child("PlatformServices").Child("SystemService")
					svc.Children = append(svc.Children, &boxstation.Node{Name: "currentCpuUsage", Type: "baja:Integer", Value: current},
						&boxstation.Node{Name: "overallCpuUsage", Type: "baja:Integer", Value: overall})
				})
			}
		}
		return 0, nil
	})
}

// pfWarned reports whether a warning of the platform section holds every needle.
func pfWarned(platform map[string]any, needles ...string) bool {
	for _, w := range platform["warnings"].([]any) {
		s, _ := w.(string)
		all := true
		for _, n := range needles {
			all = all && strings.Contains(s, n)
		}
		if all {
			return true
		}
	}
	return false
}
