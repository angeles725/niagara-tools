package golden

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestReadBinarySections pins the non-text markers: base64 and class files, empty dirs and a jar whose
// entries keep their order, their method and their bytes.
func TestReadBinarySections(t *testing.T) {
	text := "-- args --\n@DIR@/a.jar\n-- want --\nexit 0\n-- stderr --\n" +
		"-- d/e (dir) --\n-- b.bin (base64) --\nAAEC\n-- c/A.class (class 52) --\n-- n.txt (noeol) --\nx\n" +
		"-- a.jar (jar) --\n--- META-INF/ (stored) ---\n--- B.class (deflated) (class 65) ---\n" +
		"--- t.txt (stored) (noeol) ---\nhi\n--- r.bin (deflated) (base64) ---\n/w==\n--- m.xml (deflated) ---\n<m/>\n"
	p := filepath.Join(t.TempDir(), "c.txt")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Read(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Dirs) != 1 || c.Dirs[0] != "d/e" || c.Files["b.bin"] != "\x00\x01\x02" || c.Files["n.txt"] != "x" ||
		c.Files["c/A.class"] != "\xca\xfe\xba\xbe\x00\x00\x00\x34" {
		t.Fatalf("files %q dirs %q", c.Files, c.Dirs)
	}
	zr, err := zip.NewReader(bytes.NewReader([]byte(c.Files["a.jar"])), int64(len(c.Files["a.jar"])))
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name   string
		method uint16
		data   string
	}{
		{"META-INF/", zip.Store, ""}, {"B.class", zip.Deflate, "\xca\xfe\xba\xbe\x00\x00\x00\x41"},
		{"t.txt", zip.Store, "hi"}, {"r.bin", zip.Deflate, "\xff"}, {"m.xml", zip.Deflate, "<m/>\n"},
	}
	if len(zr.File) != len(want) {
		t.Fatalf("%d entries, want %d", len(zr.File), len(want))
	}
	for i, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		if w := want[i]; f.Name != w.name || f.Method != w.method || string(data) != w.data {
			t.Errorf("entry %d: %s method %d %q, want %+v", i, f.Name, f.Method, data, w)
		}
	}
}

// TestBuildJarBadBase64 makes an undecodable base64 jar entry a BuildJar error, never a jar with a short entry.
func TestBuildJarBadBase64(t *testing.T) {
	if _, err := BuildJar("--- x (base64) ---\n!!\n"); err == nil {
		t.Error("BuildJar accepted a bad base64 entry")
	}
}

// TestReadBadBase64 makes an undecodable base64 body a Read error, never an empty file.
func TestReadBadBase64(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.txt")
	if err := os.WriteFile(p, []byte("-- a.bin (base64) --\n!!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(p); err == nil {
		t.Fatal("bad base64 read without error")
	}
}
