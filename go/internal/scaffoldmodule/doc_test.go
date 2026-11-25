package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// docKitFiles is templates/doc in the small kit: files copied into the module directory (the page named by the
// registered type, @DOCTYPE@) and the map row put before the plumbing row. The fixtures' maps are in kitFiles.
func docKitFiles() map[string]string {
	tp := "templates/doc/"
	return map[string]string{
		tp + "files/ModuleA-doc/ModuleA-doc.gradle.kts":         "// Example ModuleA doc\n",
		tp + "files/ModuleA-doc/src/doc/ModuleA-@DOCTYPE@.html": "<h1>@DOCTYPE@ by Example</h1>\n",
		tp + "fragments/module-map-row.txt":                     "| help | rt | ModuleA-doc/src/doc/ModuleA-@DOCTYPE@.html |\n",
	}
}

// docKit is the small kit with the module maps the doc row is put into (the dashboard and service maps come
// with their own files; the others are added here so the other tests' file counts stay as they are).
func docKit(t *testing.T) string {
	t.Helper()
	tb := newKit(t)
	for _, f := range []string{"MinimalPan", "MinimalDriver"} {
		path := tb + "/../fixtures/" + f + "/" + f + "/MODULE-MAP.md"
		if err := os.WriteFile(path, []byte("| control | rt | x |\n| plumbing | rt | y |\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return tb
}

// --with-doc adds a -doc part; its help page is named by the type's registered name (no B): the module name
// for a logic or dashboard module, plus Network for a driver and Service for a service.
func TestWithDoc(t *testing.T) {
	tb := docKit(t)
	for typ, page := range map[string]string{"logic": "ColdBox", "dashboard": "ColdBox", "driver": "ColdBoxNetwork", "service": "ColdBoxService"} {
		out := t.TempDir()
		r := run(t, tb, "ColdBox", out, "--with-doc", "--type", typ, "--vendor", "Acme")
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		got := tree(t, filepath.Join(out, "ColdBox"))
		wantFile(t, got, "ColdBox/ColdBox-doc/ColdBox-doc.gradle.kts", "// Acme ColdBox doc\n")
		wantFile(t, got, "ColdBox/ColdBox-doc/src/doc/ColdBox-"+page+".html", "<h1>"+page+" by Acme</h1>\n")
		if m := got["ColdBox/MODULE-MAP.md"]; !strings.Contains(m, "| help | rt | ColdBox-doc/src/doc/ColdBox-"+page+".html |\n| plumbing |") {
			t.Errorf("%s: map has no help row before plumbing: %q", typ, m)
		}
	}
}

// The flag may follow or precede the valued options, and a run without it emits no -doc part.
func TestWithDocPositionAndAbsence(t *testing.T) {
	tb := docKit(t)
	out := t.TempDir()
	if r := run(t, tb, "Foo", out, "--vendor", "Acme", "--with-doc", "--target-version", "4.15"); r.code != 0 {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	if _, err := os.Stat(out + "/Foo/Foo/Foo-doc"); err != nil {
		t.Errorf("no -doc part: %v", err)
	}
	if r := run(t, tb, "Bar", out, "--vendor", "Acme"); r.code != 0 {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	if _, err := os.Stat(out + "/Bar/Bar/Bar-doc"); err == nil {
		t.Error("a run without --with-doc emitted a -doc part")
	}
}

// A library registers no type, so there is no page to name: exit 2 naming the flag, nothing created.
func TestWithDocRefusedForLibrary(t *testing.T) {
	out := t.TempDir()
	r := run(t, newKit(t), "Foo", out, "--type", "library", "--with-doc", "--vendor", "Acme")
	want := "scaffold-module: --with-doc needs a type that registers a component type; library has none\n"
	if r.code != 2 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	if _, err := os.Stat(out + "/Foo"); err == nil {
		t.Error("destination created")
	}
}

// A kit without templates/doc is exit 3 naming it, partial tree removed; a run without the flag never reads it.
func TestDocTemplateMissing(t *testing.T) {
	tb := newKit(t, "templates/doc")
	out := t.TempDir()
	r := run(t, tb, "Foo", out, "--with-doc", "--vendor", "Angeles")
	want := "scaffold-module: doc template not found: " + tb + "/../templates/doc\n" +
		"scaffold-module: removed the partial " + out + "/Foo\n"
	if r.code != 3 || r.out != "" || r.err != want {
		t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	if r := run(t, tb, "Bar", out, "--vendor", "Angeles"); r.code != 0 {
		t.Errorf("without the flag: code=%d err=%q", r.code, r.err)
	}
}

// PRESERVE: an option value is taken as typed, so a value that looks like the flag is still a value.
func TestFlagLookalikeStaysAValue(t *testing.T) {
	r := run(t, newKit(t), "Foo", t.TempDir(), "--vendor", "--with-doc")
	if r.code != 2 || r.err == "" {
		t.Errorf("code=%d err=%q", r.code, r.err)
	}
}
