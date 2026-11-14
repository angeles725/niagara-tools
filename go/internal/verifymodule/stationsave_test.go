package verifymodule

import (
	"strings"
	"testing"
)

// stationSaveRow runs the gate with --src on a Foo-rt module whose src holds files (name -> Java text) and
// returns the status and the detail of its station-save row.
func stationSaveRow(t *testing.T, files map[string]string) (status, detail string) {
	t.Helper()
	dir := t.TempDir()
	tree := map[string]string{"Foo-rt.jar": "META-INF/module.xml=<module/>"}
	for name, text := range files {
		tree["mod/Foo-rt/src/com/x/"+name] = text
	}
	write(t, dir, tree)
	_, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
	for _, ln := range strings.Split(out, "\n") {
		if f := strings.Fields(ln); len(f) >= 2 && f[1] == "station-save" {
			return f[0], strings.SplitN(ln, "@DIR@/Foo-rt.jar  ", 2)[1]
		}
	}
	t.Fatalf("no station-save row in\n%s", out)
	return "", ""
}

// TestStationSave: the row counts the classes of the profile's src that call Sys.getStation().save() and
// names each at file:line. None or one is a PASS; two or more is a WARN that never fails the gate. A call in
// a comment or in a string is no call, and a call split across lines is one.
func TestStationSave(t *testing.T) {
	const (
		src  = "@DIR@/mod/Foo-rt/src/com/x/"
		call = "class %s {\n  void tick() {\n    Sys.getStation().save();\n  }\n}\n"
	)
	class := func(name string) string { return strings.ReplaceAll(call, "%s", name) }
	for _, tc := range []struct {
		name           string
		files          map[string]string
		status, detail string
	}{
		{"none", map[string]string{"A.java": "class A {}\n"},
			"PASS", "no class calls Sys.getStation().save() under @DIR@/mod/Foo-rt/src"},
		{"one", map[string]string{"A.java": "class A {}\n", "Saver.java": class("Saver")},
			"PASS", "1 class calls Sys.getStation().save(): " + src + "Saver.java:3"},
		{"two", map[string]string{"ASaver.java": class("ASaver"), "BSaver.java": "\n" + class("BSaver")},
			"WARN", "2 classes call Sys.getStation().save(): " + src + "ASaver.java:3, " + src + "BSaver.java:4" + stationSaveRule},
		{"twice-in-one-class", map[string]string{"A.java": "class A {\n  void a() { Sys.getStation().save(); }\n  void b() { Sys.getStation().save(); }\n}\n"},
			"PASS", "1 class calls Sys.getStation().save(): " + src + "A.java:2"},
		{"comment", map[string]string{"A.java": "class A {\n  // Sys.getStation().save();\n  /* Sys.getStation().save(); */\n  /**\n   * SAVE -> Sys.getStation().save();\n   */\n  void a() {}\n}\n"},
			"PASS", "no class calls Sys.getStation().save() under @DIR@/mod/Foo-rt/src"},
		{"string", map[string]string{"A.java": "class A {\n  String s = \"Sys.getStation().save();\";\n  String u = \"x\" + \" Sys.getStation().save() \";\n}\n"},
			"PASS", "no class calls Sys.getStation().save() under @DIR@/mod/Foo-rt/src"},
		{"split-across-lines", map[string]string{"A.java": "class A {\n  void a() {\n    Sys\n      .getStation() // the running station\n      .save\n      ();\n  }\n}\n"},
			"PASS", "1 class calls Sys.getStation().save(): " + src + "A.java:3"},
		{"other-save", map[string]string{"A.java": "class A {\n  void a() { file.save(); Sys.getStation().getSlotPath(); Sys.getStation().saveX(); }\n}\n"},
			"PASS", "no class calls Sys.getStation().save() under @DIR@/mod/Foo-rt/src"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if status, detail := stationSaveRow(t, tc.files); status != tc.status || detail != tc.detail {
				t.Errorf("%s %q\nwant %s %q", status, detail, tc.status, tc.detail)
			}
		})
	}
}

// TestStationSaveStrict: two requesters never fail the gate. With and without --strict the row is the same
// WARN and the exit code is the one the same module gives with no requester at all.
func TestStationSaveStrict(t *testing.T) {
	gate := func(body string, flags ...string) (int, string) {
		dir := t.TempDir()
		write(t, dir, map[string]string{
			"Foo-rt.jar":                  "META-INF/module.xml=<module/>",
			"mod/Foo-rt/src/com/x/A.java": "class A { void a() { " + body + " } }\n",
			"mod/Foo-rt/src/com/x/B.java": "class B { void b() { " + body + " } }\n",
		})
		code, out, _ := runMain(dir, append(flags, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")...)
		for _, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "  station-save  ") {
				return code, ln
			}
		}
		t.Fatalf("no station-save row in\n%s", out)
		return 0, ""
	}
	for _, flags := range [][]string{nil, {"--strict"}} {
		base, _ := gate("", flags...)
		code, row := gate("Sys.getStation().save();", flags...)
		if !strings.HasPrefix(row, "WARN  station-save  ") || code != base {
			t.Errorf("%v: exit %d row %q, want a WARN row and exit %d (the module without a requester)", flags, code, row, base)
		}
	}
}

// TestStationSaveNoSrc: without --src, and with a --src that has no src dir for the jar, the row is a SKIP.
func TestStationSaveNoSrc(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{"Foo-rt.jar": "META-INF/module.xml=<module/>", "mod/keep.txt": ""})
	_, out, _ := runMain(dir, "@DIR@/Foo-rt.jar")
	if want := "SKIP  station-save  @DIR@/Foo-rt.jar  no --src\n"; !strings.Contains(out, want) {
		t.Errorf("no --src: want %q in\n%s", want, out)
	}
	_, out, _ = runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
	if want := "SKIP  station-save  @DIR@/Foo-rt.jar  no @DIR@/mod/Foo-rt/src\n"; !strings.Contains(out, want) {
		t.Errorf("no src dir: want %q in\n%s", want, out)
	}
}
