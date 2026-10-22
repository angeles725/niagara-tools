package jarversion

import (
	"archive/zip"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// entry is one member of a test jar; method is zip.Store or zip.Deflate (Store when zero).
type entry struct {
	name   string
	data   []byte
	method uint16
	flags  uint16
}

func classHeader(major int) []byte {
	return []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, byte(major >> 8), byte(major), 0, 0}
}

// writeJar writes a zip archive with entries in order (a raw entry when method is not Store or
// Deflate, so a test can carry pre-compressed data) and returns its path.
func writeJar(t *testing.T, dir, name string, entries ...entry) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for _, e := range entries {
		h := &zip.FileHeader{Name: e.name, Method: e.method, Flags: e.flags}
		var w interface{ Write([]byte) (int, error) }
		if e.method == zip.Store || e.method == zip.Deflate {
			w, err = zw.CreateHeader(h)
		} else {
			h.UncompressedSize64 = 8
			h.CompressedSize64 = uint64(len(e.data))
			w, err = zw.CreateRaw(h)
		}
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(e.data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCheckJar(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	var limit []entry
	for i := 0; i < 20; i++ {
		limit = append(limit, entry{name: "p/C" + strconv.Itoa(i) + ".class", data: classHeader(52)})
	}
	limit = append(limit, entry{name: "p/Late.class", data: classHeader(53)})
	// bzip2 (method 12) of CAFEBABE 0000 0035: unzip reads it, so the port must too.
	bz, err := hex.DecodeString("425a683931415926535958da9953000003480cc0000200001100100001200021800c0223addc5dc914e14241636a654c")
	if err != nil {
		t.Fatal(err)
	}
	notZip := filepath.Join(d, "notzip.jar")
	if err := os.WriteFile(notZip, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		path    string
		want    Finding
		wantHit bool
	}{
		{"Java 8 only is clean", writeJar(t, d, "j8.jar", entry{name: "Foo.class", data: classHeader(52)}), Finding{}, false},
		{"a deflated Java 9 class fails", writeJar(t, d, "j9.jar", entry{name: "a/Bar.class", data: classHeader(53), method: zip.Deflate}),
			Finding{"a/Bar.class", 53}, true},
		{"mixed: the first Java 9+ class in archive order is reported, then the jar stops",
			writeJar(t, d, "mixed.jar", entry{name: "A.class", data: classHeader(52)}, entry{name: "B.class", data: classHeader(55)},
				entry{name: "C.class", data: classHeader(61)}),
			Finding{"B.class", 55}, true},
		{"only the first 20 sampled classes are read", writeJar(t, d, "limit.jar", limit...), Finding{}, false},
		{"multi-release versions/ classes are not read", writeJar(t, d, "mr.jar", entry{name: "META-INF/versions/11/A.class", data: classHeader(55)},
			entry{name: "A.class", data: classHeader(52)}), Finding{}, false},
		{"a non-class entry is not read", writeJar(t, d, "res.jar", entry{name: "a.bin", data: classHeader(60)}), Finding{}, false},
		{"no classes at all", writeJar(t, d, "empty.jar"), Finding{}, false},
		{"a seven-byte class reports its seventh byte", writeJar(t, d, "seven.jar", entry{name: "S.class", data: classHeader(53)[1:8]}),
			Finding{"S.class", 53}, true},
		{"a six-byte class has no version", writeJar(t, d, "six.jar", entry{name: "S.class", data: classHeader(60)[:6]}), Finding{}, false},
		{"a bzip2 entry is read", writeJar(t, d, "bz.jar", entry{name: "Z.class", data: bz, method: 12}), Finding{"Z.class", 53}, true},
		{"an encrypted entry is not read (unzip cannot without a password)",
			writeJar(t, d, "enc.jar", entry{name: "E.class", data: classHeader(60), flags: 0x1}), Finding{}, false},
		{"a duplicate name reads its first record, as unzip -p did",
			writeJar(t, d, "dup.jar", entry{name: "D.class", data: classHeader(52)}, entry{name: "D.class", data: classHeader(60)}), Finding{}, false},
		// Deliberate fix: unzip -p took the entry name as a wildcard, so these read nothing and passed.
		{"a name with [ ] is read literally", writeJar(t, d, "brackets.jar", entry{name: "Foo[1].class", data: classHeader(53)}), Finding{"Foo[1].class", 53}, true},
		{"a name with a backslash is read literally", writeJar(t, d, "backslash.jar", entry{name: `back\slash.class`, data: classHeader(56)}),
			Finding{`back\slash.class`, 56}, true},
		{"a name with a control character is read", writeJar(t, d, "tab.jar", entry{name: "a\tb.class", data: classHeader(54)}), Finding{"a\tb.class", 54}, true},
		{"a file that is not a zip is clean (unzip -Z1 failed silently)", notZip, Finding{}, false},
		{"a missing file is clean", filepath.Join(d, "missing.jar"), Finding{}, false},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, hit := CheckJar(tt.path)
			if hit != tt.wantHit || got != tt.want {
				t.Errorf("CheckJar(%s) = %+v, %v; want %+v, %v", tt.path, got, hit, tt.want, tt.wantHit)
			}
		})
	}
}
