package vendorversion

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// dupJar writes a jar whose entries are all named META-INF/module.xml, one per body.
func dupJar(t *testing.T, bodies ...[]byte) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, b := range bodies {
		w, err := zw.Create(moduleXMLEntry)
		if err == nil {
			_, err = w.Write(b)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "dup.jar")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestModuleVersionFromZipBoundsTheCumulativeRead: the bound is on all META-INF/module.xml entries together,
// not on each, so many duplicates cannot make the reader hold more than maxModuleXML; the entries under the
// bound are all read, in order.
func TestModuleVersionFromZipBoundsTheCumulativeRead(t *testing.T) {
	t.Parallel()
	filler := bytes.Repeat(append(bytes.Repeat([]byte("x"), 1023), '\n'), maxModuleXML/1024) // exactly the bound
	tag := []byte("<module name=\"X\" vendorVersion=\"1.2.3\">\n")
	if got := ModuleVersionFromZip(dupJar(t, []byte("<?xml?>\n"), tag)); got != "1.2.3" {
		t.Errorf("two small entries: %q, want the version of the second", got)
	}
	if got := ModuleVersionFromZip(dupJar(t, filler, tag)); got != "" {
		t.Errorf("a second entry after a bound-sized first: %q, want empty (the bound is spent)", got)
	}
}
