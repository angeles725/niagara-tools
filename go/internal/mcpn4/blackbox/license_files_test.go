package blackbox

// n4_license_info with --niagara-home: when each licence was generated and its version, read
// from the root element of the licence files of the local installation and nothing else of them.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// licFile is a licence file whose root element has attrs; its body holds a serial number, a
// feature and a signature that must never reach a reply.
func licFile(attrs string) string {
	return `<license ` + attrs + ` serialNumber="LICSERIAL">` + "\n" + `  <feature name="about" owner="LICOWNERX"/>` +
		"\n  <signature>LICSIGNATUREX</signature>\n</license>\n"
}

// licHome is a Niagara installation directory holding files (name: content) as its licences.
func licHome(t *testing.T, files map[string]string) string {
	t.Helper()
	home := t.TempDir()
	dir := filepath.Join(home, "security", "licenses")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return home
}

// licFilesRig is a licence rig (licences A and B, both dated 2027-03-31, host Lab-0000) whose
// server was started with --niagara-home for the station, over files.
func licFilesRig(t *testing.T, files map[string]string) *Rig {
	t.Helper()
	home := licHome(t, files)
	r, _ := licRigWith(t, licNow, licSmall("2027-03-31", "2027-03-31"), func(o *Options) {
		o.NiagaraHomes = map[string]string{StationName: home}
	})
	return r
}

const licAttrs = `vendor="Tridium" expiration="2027-03-31" hostId="Lab-0000" version="4.15" generated="2026-10-12"`

func TestLicenseInfoAddsGeneratedAndVersionFromTheLocalLicenceFiles(t *testing.T) {
	r := licFilesRig(t, map[string]string{"A.license": licFile(licAttrs),
		"B.license": licFile(`vendor="Other" expiration="2027-03-31" hostId="Lab-0000" version="4.14" generated="2025-01-02"`),
		"notes.txt": "not a licence"})
	res := r.Srv.Call("n4_license_info", nil)
	out := res.Structured
	want := map[string]any{"name": "A.license", "expires": "2027-03-31", "valid": true, "days_left": 136.0,
		"generated": "2026-10-12", "version": "4.15"}
	if got := licNamed(t, out["licenses"], "A.license"); !reflect.DeepEqual(got, want) {
		t.Errorf("A = %v", got)
	}
	if got := licNamed(t, out["licenses"], "B.license"); got["generated"] != "2025-01-02" || got["version"] != "4.14" {
		t.Errorf("B = %v", got)
	}
	if _, noted := out["generated_note"]; noted {
		t.Errorf("generated_note = %v", out["generated_note"])
	}
	for _, secret := range []string{"LICSIGNATUREX", "LICSERIAL", "LICOWNERX"} {
		if strings.Contains(res.Text, secret) {
			t.Errorf("the reply holds %s of a licence file", secret)
		}
	}
}

func TestLicenseInfoWithoutANiagaraHomeOmitsGeneratedAndSaysWhy(t *testing.T) {
	r, _ := licRig(t, licNow, licSmall("2027-03-31"))
	out := r.Srv.OK("n4_license_info", nil)
	got := licNamed(t, out["licenses"], "A.license")
	_, generated := got["generated"]
	_, version := got["version"]
	if note, _ := out["generated_note"].(string); generated || version || !strings.Contains(note, "--niagara-home "+StationName+"=") {
		t.Errorf("A = %v, generated_note = %q", got, note)
	}
}

func TestLicenseInfoRefusesLicenceFilesOfAnotherHost(t *testing.T) {
	r := licFilesRig(t, map[string]string{"A.license": licFile(strings.Replace(licAttrs, "Lab-0000", "Lab-9999", 1)),
		"B.license": licFile(licAttrs)})
	out := r.Srv.OK("n4_license_info", nil)
	if _, has := licNamed(t, out["licenses"], "A.license")["generated"]; has {
		t.Errorf("a file of another host was read into the reply: %v", out["licenses"])
	}
	note, _ := out["generated_note"].(string)
	if licNamed(t, out["licenses"], "B.license")["generated"] != "2026-10-12" || !strings.Contains(note, "A.license") ||
		!strings.Contains(note, "another host") || strings.Contains(note, "Lab-9999") {
		t.Errorf("licenses = %v, note = %q", out["licenses"], note)
	}
}

func TestLicenseInfoNamesTheLicenceFilesItCouldNotRead(t *testing.T) {
	r := licFilesRig(t, map[string]string{"A.license": "<license generated=\"2026-10-12\" LICSIGNATUREX",
		"C.license": licFile(licAttrs)})
	res := r.Srv.Call("n4_license_info", nil)
	note, _ := res.Structured["generated_note"].(string)
	if !strings.Contains(note, "A.license") || !strings.Contains(note, "B.license") || strings.Contains(res.Text, "LICSIGNATUREX") {
		t.Errorf("generated_note = %q", note)
	}
	for _, name := range []string{"A.license", "B.license"} {
		if _, has := licNamed(t, res.Structured["licenses"], name)["generated"]; has {
			t.Errorf("%s has a generated date", name)
		}
	}
	doctype := licFilesRig(t, map[string]string{"A.license": "<!DOCTYPE license [<!ENTITY x SYSTEM 'file:///etc/hostname'>]>" + licFile(licAttrs),
		"B.license": licFile(licAttrs)})
	out := doctype.Srv.OK("n4_license_info", nil)
	if _, has := licNamed(t, out["licenses"], "A.license")["generated"]; has || licNamed(t, out["licenses"], "B.license")["version"] != "4.15" {
		t.Errorf("a file with a DOCTYPE: %v", out["licenses"])
	}
	missing := licFilesRig(t, nil)
	if err := os.RemoveAll(filepath.Join(missing.Opts.NiagaraHomes[StationName], "security")); err != nil {
		t.Fatal(err)
	}
	if note, _ := missing.Srv.OK("n4_license_info", nil)["generated_note"].(string); !strings.Contains(note, "security/licenses") {
		t.Errorf("no licences directory: %q", note)
	}
}

func TestLicenseInfoReadsTheMaintenanceEndDateALicenceFileStates(t *testing.T) {
	r := licFilesRig(t, map[string]string{"A.license": licFile(licAttrs + ` maintenanceExpiration="2026-12-15"`),
		"B.license": licFile(licAttrs + ` maintenanceExpiration="2027-03-31"`)})
	got := shxSection(t, r.Srv.OK("n4_license_info", nil), "maintenance")
	if got["expires"] != "2026-12-15" || got["days_left"] != 30.0 || got["license"] != "A.license" {
		t.Errorf("maintenance = %v", got)
	}
	none := licFilesRig(t, map[string]string{"A.license": licFile(licAttrs), "B.license": licFile(licAttrs)})
	got = shxSection(t, none.Srv.OK("n4_license_info", nil), "maintenance")
	if note, _ := got["note"].(string); got["expires"] != nil || !strings.Contains(note, "no maintenanceExpiration") {
		t.Errorf("no attribute: %v", got)
	}
}

// A licence whose file is there but was not read is told apart from one with no file, with the
// reason; a local file that matches no licence of the page is never named.
func TestLicenseInfoSaysWhyALicenceFileWasSkippedAndNeverNamesAnUnrelatedFile(t *testing.T) {
	home := licHome(t, map[string]string{"B.LICENSE": licFile(licAttrs), "zother.license": "junk LICSIGNATUREX", "Extra.license": licFile(licAttrs)})
	outside := filepath.Join(t.TempDir(), "outside.xml")
	if err := os.WriteFile(outside, []byte(licFile(licAttrs)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(home, "security", "licenses", "A.license")); err != nil {
		t.Fatal(err)
	}
	r, _ := licRigWith(t, licNow, licSmall("2027-03-31", "2027-03-31", "2027-03-31"), func(o *Options) {
		o.NiagaraHomes = map[string]string{StationName: home}
	})
	res := r.Srv.Call("n4_license_info", nil)
	note, _ := res.Structured["generated_note"].(string)
	for _, want := range []string{"not a regular file (a link, a directory or a device): A.license", "letter case (the extension must be .license): B.license",
		"no file of that name under security/licenses: C.license"} {
		if !strings.Contains(note, want) {
			t.Errorf("generated_note lacks %q: %q", want, note)
		}
	}
	for _, never := range []string{"zother", "Extra", "LICSIGNATUREX", "outside.xml"} {
		if strings.Contains(res.Text, never) {
			t.Errorf("the reply names %s", never)
		}
	}
	for _, name := range []string{"A.license", "B.license", "C.license"} {
		if _, has := licNamed(t, res.Structured["licenses"], name)["generated"]; has {
			t.Errorf("%s has a generated date", name)
		}
	}
}
