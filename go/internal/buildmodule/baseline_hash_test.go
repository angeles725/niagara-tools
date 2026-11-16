package buildmodule

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

// jarOf writes a jar holding module.xml at version plus the given payload entries (name, content) in order.
func (f *fixture) jarOf(path, version string, entries ...[2]string) []byte {
	f.t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	all := append([][2]string{{"META-INF/module.xml", `<module name="X" vendor="A" vendorVersion="` + version + `">` + "\n</module>\n"}}, entries...)
	for _, e := range all {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: e[0], Method: zip.Store})
		if err == nil {
			_, err = w.Write([]byte(e[1]))
		}
		if err != nil {
			f.t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		f.t.Fatal(err)
	}
	f.write(path, buf.String())
	return buf.Bytes()
}

// TestContentHashCoversNamesAndSizes: the same bytes moved between entries, or under another name, are a
// change of what ships; at an unchanged version the gate reports drift.
func TestContentHashCoversNamesAndSizes(t *testing.T) {
	for name, built := range map[string][][2]string{
		"bytes moved between entries": {{"com/x/A.class", "x"}, {"com/x/B.class", "yz"}},
		"entry renamed":               {{"com/x/A.class", "xy"}, {"com/x/C.class", "z"}},
	} {
		f := newFixture(t)
		f.jarOf(f.deployed("rt"), "1.0.0", [2]string{"com/x/A.class", "xy"}, [2]string{"com/x/B.class", "z"})
		f.jarOf(f.built("rt"), "1.0.0", built...)
		f.wants(name, f.build(), ExitDrift, "Foo-rt shipped bytes changed", "build.sh: deployed-baseline drift detected")
	}
}

// TestUnreadableEntryCannotBeVerified: an entry whose bytes fail their CRC is "cannot verify" (exit 52), even
// when the deployed jar is the same damaged file; a file that is not a zip, a zip with no entry, or one with
// two entries under one name (unzip -p printed both; which one a class loader reads is its choice), too.
func TestUnreadableEntryCannotBeVerified(t *testing.T) {
	damaged := func(f *fixture, path string) {
		raw := f.jarOf(path, "1.0.0", [2]string{"com/x/A.class", "payload"})
		f.write(path, strings.Replace(string(raw), "payload", "pAyload", 1))
	}
	empty := func(f *fixture, path string) {
		var buf bytes.Buffer
		if err := zip.NewWriter(&buf).Close(); err != nil {
			f.t.Fatal(err)
		}
		f.write(path, buf.String())
	}
	for name, make := range map[string]func(*fixture, string){
		"an entry fails its CRC": damaged,
		"not a zip":              func(f *fixture, path string) { f.write(path, "not a jar\n") },
		"no entry at all":        empty,
		"two entries under one name": func(f *fixture, path string) {
			f.jarOf(path, "1.0.0", [2]string{"com/x/A.class", "one"}, [2]string{"com/x/A.class", "two"})
		},
	} {
		f := newFixture(t)
		make(f, f.deployed("rt"))
		make(f, f.built("rt"))
		f.wants(name, f.build(), ExitDriftUnknown,
			"build.sh: FAIL — drift gate cannot verify Foo-rt: "+whyUnlistable+".\n",
			"build.sh: deployed-baseline drift gate could not verify the jar(s) above")
	}
}

// TestContentHashSizeChanges: with the names unchanged, an entry that grows is a change of what ships; and two
// jars whose names and bytes concatenate to one stream still differ, because each entry's size is hashed.
func TestContentHashSizeChanges(t *testing.T) {
	const a, b = "com/x/A.class", "com/x/B.class"
	type jar = [][2]string
	for name, c := range map[string]struct{ deployed, built jar }{
		"one entry grows": {jar{{a, "xy"}, {b, "z"}}, jar{{a, "xyq"}, {b, "z"}}},
		"sizes differ, names and bytes concatenate equal": {
			jar{{a, "p" + b + "\x00q"}, {b, "r"}},
			jar{{a, "p"}, {b, "q" + b + "\x00r"}}},
	} {
		f := newFixture(t)
		f.jarOf(f.deployed("rt"), "1.0.0", c.deployed...)
		f.jarOf(f.built("rt"), "1.0.0", c.built...)
		f.wants(name, f.build(), ExitDrift, "Foo-rt shipped bytes changed", "build.sh: deployed-baseline drift detected")
	}
}

// TestIdenticalTreesHashEqual: the same entries with the same bytes are unchanged whatever order the jar
// stores them in, at an unchanged version too.
func TestIdenticalTreesHashEqual(t *testing.T) {
	x, y := [2]string{"com/x/A.class", "xy"}, [2]string{"com/x/B.class", "z"}
	for name, built := range map[string][][2]string{"same order": {x, y}, "stored in another order": {y, x}} {
		f := newFixture(t)
		f.signs()
		f.jarOf(f.deployed("rt"), "1.0.0", x, y)
		f.jarOf(f.built("rt"), "1.0.0", built...)
		f.wants(name, f.build(), ExitOK)
		if f.err != "" {
			t.Errorf("%s: stderr %q", name, f.err)
		}
	}
}
