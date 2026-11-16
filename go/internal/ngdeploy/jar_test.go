package ngdeploy

import (
	"archive/zip"
	"bytes"
	"sort"
)

// zipJar writes a real jar: a zip holding the entries (name to content) in name order. A name ending in
// "/" is a directory entry.
func (f *fixture) zipJar(path string, entries map[string]string) {
	f.t.Helper()
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			f.t.Fatal(err)
		}
		w.Write([]byte(entries[name]))
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	f.write(path, buf.String())
}

// signedBy is a jar's entries with one complete signature per name: META-INF/<NAME>.SF and its block file.
func signedBy(names ...string) map[string]string {
	entries := map[string]string{"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\r\n\r\n", "com/acme/BDemo.class": "class"}
	for _, name := range names {
		entries["META-INF/"+name+".SF"] = "Signature-Version: 1.0\r\n"
		entries["META-INF/"+name+".RSA"] = "block"
	}
	return entries
}

// declares writes a jar with one entry that declares size bytes of content and holds none: what the
// directory of a zip says is all the precheck reads before it decides how much it will read.
func (f *fixture) declares(path, name string, size uint64) {
	f.t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	if _, err := zw.CreateRaw(&zip.FileHeader{Name: name, Method: zip.Deflate, UncompressedSize64: size, CompressedSize64: 0}); err != nil {
		f.t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	f.write(path, buf.String())
}
