package vendorfloor

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageText = "usage: lint-vendor-floor.sh <rc-dir> [--strict]\n"

// call runs the lint with vars as its whole environment and the shipped table.
func call(vars map[string]string, args ...string) (code int, out, errs string) {
	return callWith(Default, vars, args...)
}

func callWith(apis APIs, vars map[string]string, args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = run(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(name string) string { return vars[name] }}, apis)
	return code, o.String(), e.String()
}

// needNode skips the test when node is not installed and returns the environment that finds it.
func needNode(t *testing.T) map[string]string {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not installed: the acorn driver cannot run")
	}
	return map[string]string{"KIT_NODE": node}
}

// needAcorn skips the test when node or acorn is missing and returns the environment that finds both:
// KIT_ACORN, else the toolbelt install, else an acorn node resolves by itself.
func needAcorn(t *testing.T) map[string]string {
	t.Helper()
	vars := needNode(t)
	toolbelt, _ := filepath.Abs("../../../build-n4-module-kit/toolbelt")
	switch {
	case os.Getenv("KIT_ACORN") != "":
		vars["KIT_ACORN"] = os.Getenv("KIT_ACORN")
	case isDir(toolbelt + bundledAcorn):
		vars["LINT_VENDOR_FLOOR_TOOLBELT"] = toolbelt
	case exec.Command(vars["KIT_NODE"], "-e", `require.resolve("acorn")`).Run() != nil:
		t.Skip("acorn not installed (npm install --prefix build-n4-module-kit/toolbelt/eslint, or set KIT_ACORN)")
	}
	return vars
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// rc makes an rc dir whose vendor/ holds files (name -> source) and returns its path.
func rc(t *testing.T, files map[string]string) string {
	t.Helper()
	d := t.TempDir()
	for name, src := range files {
		write(t, d, "vendor/"+name, src)
	}
	return d
}

func check(t *testing.T, name string, code int, out, errs string, wantCode int, wantOut, wantErrs string) {
	t.Helper()
	if code != wantCode || out != wantOut || errs != wantErrs {
		t.Errorf("%s:\n exit %d, want %d\n stdout %q\n   want %q\n stderr %q\n   want %q", name, code, wantCode, out, wantOut, errs, wantErrs)
	}
}

func TestUsageAndRootErrors(t *testing.T) {
	t.Parallel()
	d := t.TempDir()
	file := write(t, d, "f.js", "")
	cases := []struct {
		name string
		args []string
		errs string
	}{
		{"no argument", nil, usageText},
		{"only --strict", []string{"--strict"}, usageText},
		{"an unknown flag", []string{d, "--bogus"}, usageText},
		{"a lone dash", []string{d, "-"}, usageText},
		{"a double dash", []string{"--", d}, usageText},
		{"two directories", []string{d, d}, usageText},
		{"an empty argument alone", []string{""}, usageText},
		{"an empty argument after the directory", []string{d, ""}, usageText},
		{"a missing directory", []string{d + "/none"}, "lint-vendor-floor: not a directory: " + d + "/none\n"},
		{"a file", []string{file}, "lint-vendor-floor: not a directory: " + file + "\n"},
	}
	for _, c := range cases {
		code, out, errs := call(nil, c.args...)
		check(t, c.name, code, out, errs, ExitEnv, "", c.errs)
	}
}

// Nothing to check needs no tool: node is forced missing and the exit is still 0.
func TestNothingToCheckIsCleanWithoutAnyTool(t *testing.T) {
	t.Parallel()
	missing := map[string]string{"KIT_NODE": "/nonexistent/node"}
	none := t.TempDir()
	code, out, errs := call(missing, none)
	check(t, "no vendor/", code, out, errs, ExitOK, "", "lint-vendor-floor: no vendor/ under "+none+" — nothing to check\n")

	docs := rc(t, map[string]string{"THIRD-PARTY.md": "", "lib.css": ""})
	code, out, errs = call(missing, "--strict", docs)
	check(t, "no *.js", code, out, errs, ExitOK, "", "lint-vendor-floor: no *.js under "+docs+"/vendor — nothing to check\n")

	// a leading empty argument leaves the directory unset, so the next one takes its place
	code, out, errs = call(missing, "", none)
	check(t, "empty argument first", code, out, errs, ExitOK, "", "lint-vendor-floor: no vendor/ under "+none+" — nothing to check\n")
}

// Mutation: VF3 -- treat a missing node as a clean pass (exit 0) so the SKIP row and exit 4 vanish
func TestMissingNodeIsASkipRowNeverAPass(t *testing.T) {
	t.Parallel()
	d := rc(t, map[string]string{"lib.js": "var a = 1;\n"})
	want := "SKIP  vendor-floor  " + d + "  unavailable: node not found (install Node, or set KIT_NODE)\n"
	code, out, errs := call(map[string]string{"KIT_NODE": "/nonexistent/node"}, d)
	check(t, "KIT_NODE missing", code, out, errs, ExitUnavailable, want, "")
	code, out, errs = call(map[string]string{"PATH": d}, d, "--strict")
	check(t, "no node on PATH", code, out, errs, ExitUnavailable, want, "")
}

func TestMissingAcornIsASkipRowNeverAPass(t *testing.T) {
	t.Parallel()
	vars := needNode(t)
	vars["KIT_ACORN"] = "/nonexistent/acorn"
	d := rc(t, map[string]string{"lib.js": "x ??= 1;\n"})
	code, out, errs := call(vars, d)
	check(t, "KIT_ACORN missing", code, out, errs, ExitUnavailable,
		"SKIP  vendor-floor  "+d+"  unavailable: acorn not resolvable from '/nonexistent/acorn' (npm install --prefix toolbelt/eslint, or set KIT_ACORN)\n", "")
}

// A node that ends without the driver's terminal record checked nothing: an env fault, never exit 0.
func TestNodeThatEndsWithoutAResultIsAnEnvFault(t *testing.T) {
	t.Parallel()
	d := rc(t, map[string]string{"lib.js": "var a = 1;\n"})
	fake := write(t, t.TempDir(), "node", "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(fake, 0o755); err != nil {
		t.Fatal(err)
	}
	code, out, errs := call(map[string]string{"KIT_NODE": fake}, d)
	check(t, "fake node", code, out, errs, ExitEnv, "", "lint-vendor-floor: env fault: "+fake+" ended without a result (no terminal record)\n")
}

func TestSharedMethodNameIsAnEnvFault(t *testing.T) {
	t.Parallel()
	vars := needNode(t)
	d := rc(t, map[string]string{"lib.js": "var a = [1].at(0);\n"})
	apis := APIs{Member: map[string]int{"at": 92}, Qualified: map[string]int{"Array.at": 92}}
	code, out, errs := callWith(apis, vars, d)
	check(t, "shared name", code, out, errs, ExitEnv, "",
		"lint-vendor-floor: env fault: MEMBER and QUALIFIED share the method name 'at' (Array.at)\n")
}
