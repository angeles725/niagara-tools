package verifymodule

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDriver is the merged DR2 scaffold every driver check is pinned on: it must pass clean, and each
// negative case is a copy of it with one doctrine rule broken.
const fixtureDriver = "../../../build-n4-module-kit/fixtures/MinimalDriver/MinimalDriver"

const (
	fxRt  = "mod/MinimalDriver-rt/src/com/angeles/MinimalDriver/"
	fxWb  = "mod/MinimalDriver-wb/src/com/angeles/MinimalDriver/wb/"
	fxJar = "MinimalDriver-rt.jar=META-INF/module.xml=<module/>"
)

// copyFixture copies the MinimalDriver module dir to <dir>/mod and returns dir.
func copyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	err := filepath.WalkDir(fixtureDriver, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixtureDriver, p)
		dst := filepath.Join(dir, "mod", rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// edit replaces old with new in <dir>/<rel>; the case is void when old is absent, so that is fatal.
func edit(t *testing.T, dir, rel, old, new string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q: the fixture changed, so this negative case no longer breaks anything", rel, old)
	}
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), old, new, -1)), 0o644); err != nil {
		t.Fatal(err)
	}
}

// driverRows runs the gate with --src on the MinimalDriver-rt stub jar and returns its driver-* rows.
func driverRows(t *testing.T, dir string) []string {
	t.Helper()
	write(t, dir, map[string]string{strings.SplitN(fxJar, "=", 2)[0]: strings.SplitN(fxJar, "=", 2)[1]})
	_, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/MinimalDriver-rt.jar")
	var rows []string
	for _, ln := range strings.Split(out, "\n") {
		if f := strings.Fields(ln); len(f) >= 2 && strings.HasPrefix(f[1], "driver-") {
			rows = append(rows, ln)
		}
	}
	return rows
}

// wantRow fails unless one row has the status and check and contains every part.
func wantRow(t *testing.T, rows []string, status, check string, parts ...string) {
	t.Helper()
	for _, r := range rows {
		f := strings.Fields(r)
		if f[0] != status || f[1] != check {
			continue
		}
		ok := true
		for _, p := range parts {
			ok = ok && strings.Contains(r, p)
		}
		if ok {
			return
		}
	}
	t.Errorf("no %s %s row containing %q in\n%s", status, check, parts, strings.Join(rows, "\n"))
}

// TestDriverFixtureClean: the DR2 scaffold passes every driver check with no FAIL or WARN row, and every
// check prints a row.
func TestDriverFixtureClean(t *testing.T) {
	rows := driverRows(t, copyFixture(t))
	for _, r := range rows {
		if s := strings.Fields(r)[0]; s == "FAIL" || s == "WARN" {
			t.Errorf("fixture row %s", r)
		}
	}
	for _, c := range driverChecks {
		found := false
		for _, r := range rows {
			found = found || strings.Fields(r)[1] == c
		}
		if !found {
			t.Errorf("no %s row on the fixture in\n%s", c, strings.Join(rows, "\n"))
		}
	}
}

// TestDriverNotDriver: a module with no driver class gets no driver row, so every recorded golden (none
// holds a driver) stays byte-identical; a driver base named only in a comment or a string is no driver.
func TestDriverNotDriver(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, map[string]string{
		"Foo-rt.jar":                  "META-INF/module.xml=<module/>",
		"mod/Foo-rt/src/com/x/A.java": "// class BA extends BNNetwork {}\nclass BB extends BComponent { String s = \"class BC extends BNDevice {\"; }\n",
	})
	_, out, _ := runMain(dir, "--src", "@DIR@/mod", "@DIR@/Foo-rt.jar")
	if strings.Contains(out, "  driver-") {
		t.Errorf("non-driver module got driver rows:\n%s", out)
	}
	if _, out, _ = runMain(dir, "@DIR@/Foo-rt.jar"); strings.Contains(out, "  driver-") {
		t.Errorf("no --src run got driver rows:\n%s", out)
	}
}

// TestDriverScaffold: lint-driver-scaffold runs on every driver module; its FAIL rows fail the gate and
// its WARN rows warn.
func TestDriverScaffold(t *testing.T) {
	dir := copyFixture(t)
	edit(t, dir, fxRt+"BMinimalDriverDevice.java", "      if (answer.isError()) pingFail(\"device error \" + answer.status);\n      else pingOk();",
		"      pingOk();")
	edit(t, dir, fxRt+"BMinimalDriverDevice.java", "    if (!hasAddress())\n    {\n      pingFail(\"ipAddress and udpPort are not set\");\n      return;\n    }\n    try\n    {\n      Frame answer = minimalDriverNetwork().send(peer(), MinimalDriverCodec.CMD_PING, 0, 0);\n      pingOk();\n    }\n    catch (Exception e)\n    {\n      pingFail(String.valueOf(e.getMessage() != null ? e.getMessage() : e));\n    }",
		"    pingOk();")
	edit(t, dir, fxRt+"MinimalDriverCodec.java", "package com.angeles.MinimalDriver;", "package com.angeles.MinimalDriver;\n// TODO add poll support")
	rows := driverRows(t, dir)
	wantRow(t, rows, "FAIL", "driver-scaffold", "BMinimalDriverDevice.java:", "only calls pingOk()")
	wantRow(t, rows, "WARN", "driver-scaffold", "MinimalDriverCodec.java:", "wizard stub")
}
