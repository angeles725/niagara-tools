package xmlsrc

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func writeZip(t *testing.T, members map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range members {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.bog")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestOpenZippedBog(t *testing.T) {
	path := writeZip(t, map[string]string{"file.xml": "<bajaObjectGraph>\n <p n=\"a$20b\"/>\n</bajaObjectGraph>"})
	d, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	e := d.Elements()
	if d.Path != path || len(e) != 2 || e[1].Line != 2 || e[1].Col != 2 {
		t.Fatalf("Open(zip) = %q %+v", d.Path, e)
	}
}

func TestOpenZippedBogErrors(t *testing.T) {
	malformed := writeZip(t, map[string]string{"file.xml": "<b>\n<p></b>"})
	noMember := writeZip(t, map[string]string{"other.xml": "<b/>"})
	corrupt := filepath.Join(t.TempDir(), "corrupt.bog")
	if err := os.WriteFile(corrupt, []byte("PK\x03\x04 not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ path, want string }{
		{malformed, malformed + ":2:4: element <p> opened at 2:1 closed by </b>"},
		{noMember, noMember + ": zip archive has no file.xml member"},
		{corrupt, corrupt + ": corrupt zip archive: zip: not a valid zip file"},
	} {
		d, err := Open(c.path)
		if d != nil || err == nil || err.Error() != c.want {
			t.Errorf("Open(%s) = %v, %v\n want error %q", filepath.Base(c.path), d, err, c.want)
		}
	}
}

func TestOpenZippedBogCap(t *testing.T) {
	path := writeZip(t, map[string]string{"file.xml": "<bajaObjectGraph/>"})
	defer func(old int) { maxMember = old }(maxMember)
	maxMember = 4
	if d, err := Open(path); d != nil || err == nil || err.Error() != path+": file.xml inflates past 4 bytes" {
		t.Fatalf("Open over the cap = %v, %v", d, err)
	}
}
