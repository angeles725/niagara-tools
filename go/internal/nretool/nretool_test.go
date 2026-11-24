package nretool

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Output of nre.exe devkit:com.tridium.devkit.util.DependencyChecker UmbrellaDashboard on a 4.14 install (CRLF, a
// localized boot log first), trimmed.
const depsOut = "INFO [nre] Launching Niagara Runtime Environment\r\n" +
	"INFORMACI\xd3N [15:41:18 23-nov-26 CST][nre] Booting\r\n" +
	"INFORMACI\xd3N [15:42:43 23-nov-26 CST][sys] Niagara runtime booted (\"C:\\Niagara\")\r\n" +
	"\r\nDependencies for UmbrellaDashboard: \r\n\r\n    baja\r\n    bql-rt\r\n    web-rt\r\n"

// fakeHome is a Niagara home whose bin/<exe> is a stub that records its arguments and prints out, exiting code.
func fakeHome(t *testing.T, exe, out string, code int) (home, argsFile string) {
	t.Helper()
	home = t.TempDir()
	argsFile = filepath.Join(home, "args.txt")
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "out.txt"), []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
	stub := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > '" + argsFile + "'\ncat '" + filepath.Join(home, "out.txt") +
		"'\nexit " + string(rune('0'+code)) + "\n"
	if err := os.WriteFile(filepath.Join(home, "bin", exe), []byte(stub), 0o755); err != nil {
		t.Fatal(err)
	}
	return home, argsFile
}

func run(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

func TestDepsPrintsOneModulePartPerLine(t *testing.T) {
	for _, exe := range []string{"nre.exe", "nre"} {
		home, argsFile := fakeHome(t, exe, depsOut, 0)
		code, out, errs := run("deps", home, "UmbrellaDashboard")
		if code != 0 || out != "baja\nbql-rt\nweb-rt\n" {
			t.Fatalf("%s: exit %d, out %q, err %q", exe, code, out, errs)
		}
		got, _ := os.ReadFile(argsFile)
		want := filepath.Join(home, "bin") + "\ndevkit:com.tridium.devkit.util.DependencyChecker\nUmbrellaDashboard\n"
		if string(got) != want {
			t.Errorf("%s: nre ran as %q, want %q", exe, got, want)
		}
	}
}

func TestDepsOfAModuleTheInstallDoesNotHold(t *testing.T) {
	home, _ := fakeHome(t, "nre.exe", "INFO [nre] Booting\r\njava.lang.NullPointerException\r\n\tat x\r\n", 249%10)
	code, out, errs := run("deps", home, "NoSuchModule")
	if code != 1 || out != "" || !strings.Contains(errs, "nre-tool: no module NoSuchModule in "+home) ||
		!strings.Contains(errs, "java.lang.NullPointerException") {
		t.Errorf("exit %d, out %q, err %q", code, out, errs)
	}
}

func TestRawPassesTheWholeOutputThrough(t *testing.T) {
	home, _ := fakeHome(t, "nre.exe", depsOut, 0)
	code, out, _ := run("deps", "--raw", home, "UmbrellaDashboard")
	if code != 0 || out != depsOut {
		t.Errorf("exit %d, out %q", code, out)
	}
}

func TestUsageAndEnvironmentFaults(t *testing.T) {
	empty := t.TempDir()
	for _, c := range []struct {
		args []string
		want string
	}{
		{nil, "usage: nre-tool deps [--raw] <niagara_home> <module>"},
		{[]string{"frobnicate", empty, "x"}, "nre-tool: unknown tool frobnicate (known: deps, list, palette, recompile)"},
		{[]string{"deps", empty}, "usage: nre-tool deps"},
		{[]string{"deps", empty, "x"}, "nre-tool: no bin/nre.exe or bin/nre under " + empty},
		{[]string{"deps", empty, "a b"}, "nre-tool: bad module name \"a b\""},
	} {
		code, out, errs := run(c.args...)
		if code != 3 || out != "" || !strings.Contains(errs, c.want) {
			t.Errorf("%q: exit %d, out %q, err %q", c.args, code, out, errs)
		}
	}
}
