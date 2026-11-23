package vendorversion

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// moduleXMLs are META-INF/module.xml texts (the cases the retired vv_module_version answered).
var moduleXMLs = []struct{ name, xml, want string }{
	{"the root tag's own version, not the baja floor", `<module name="X" vendor="V" vendorVersion="1.3"><dependencies><dependency name="baja" vendorVersion="4.13"/></dependencies></module>`, "1.3"},
	{"three parts and a suffix", `<module name="X" vendorVersion="2.10.2-rc"></module>`, "2.10.2-rc"},
	{"a dependency floor only", `<module name="X"><dependencies><dependency name="baja" vendorVersion="4.13"/></dependencies></module>`, ""},
	{"one component is not a module version", `<module name="X" vendorVersion="7"></module>`, ""},
	{"the tag on a later line", "<?xml version=\"1.0\"?>\n<!-- c -->\n<module name=\"X\"\n  vendorVersion=\"9.9\">\n<module name=\"Y\" vendorVersion=\"4.5.6\">\n", "4.5.6"},
	{"the first matching line wins", "<module name=\"A\" vendorVersion=\"1.0\">\n<module name=\"B\" vendorVersion=\"2.0\">\n", "1.0"},
	{"the last attribute of one tag", `<module vendorVersion="1.0" bajaVersion="0" vendorVersion="3.4">`, "3.4"},
	{"a non-numeric version", `<module name="X" vendorVersion="v1.0">`, ""},
	{"empty", "", ""},
}

// writeJar writes a jar holding xml as META-INF/module.xml and returns its path.
func writeJar(t *testing.T, xml string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "m.jar")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	w, err := zw.Create("META-INF/module.xml")
	if err == nil {
		_, err = w.Write([]byte(xml))
	}
	if err == nil {
		err = zw.Close()
	}
	if err == nil {
		err = f.Close()
	}
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestModuleXMLVersion(t *testing.T) {
	t.Parallel()
	for _, c := range moduleXMLs {
		if got := ModuleXMLVersion(c.xml); got != c.want {
			t.Errorf("%s: ModuleXMLVersion = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestModuleVersionReadsTheJar(t *testing.T) {
	t.Parallel()
	for _, c := range moduleXMLs {
		if got := ModuleVersion(writeJar(t, c.xml)); got != c.want {
			t.Errorf("%s: ModuleVersion = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestModuleVersionFromZipAgreesWithUnzip: the archive/zip reader answers every case the unzip reader does.
func TestModuleVersionFromZipAgreesWithUnzip(t *testing.T) {
	t.Parallel()
	for _, c := range moduleXMLs {
		if got := ModuleVersionFromZip(writeJar(t, c.xml)); got != c.want {
			t.Errorf("%s: ModuleVersionFromZip = %q, want %q", c.name, got, c.want)
		}
	}
	notAJar := filepath.Join(t.TempDir(), "x.jar")
	if err := os.WriteFile(notAJar, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, jar := range []string{notAJar, filepath.Join(t.TempDir(), "missing.jar")} {
		if got := ModuleVersionFromZip(jar); got != "" {
			t.Errorf("ModuleVersionFromZip(%s) = %q, want empty", jar, got)
		}
	}
}

func TestModuleVersionOfAnUnreadableJarIsEmpty(t *testing.T) {
	t.Parallel()
	notAJar := filepath.Join(t.TempDir(), "x.jar")
	if err := os.WriteFile(notAJar, []byte("not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, jar := range []string{notAJar, filepath.Join(t.TempDir(), "missing.jar")} {
		if got := ModuleVersion(jar); got != "" {
			t.Errorf("ModuleVersion(%s) = %q, want empty", jar, got)
		}
	}
}
