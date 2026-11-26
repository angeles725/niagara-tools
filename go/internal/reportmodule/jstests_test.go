package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// jsRun runs `report-module --lint-only` over a module root holding A-ux with the given files (path below A-ux ->
// body), a toolbelt of no-op members and node (a #!/bin/sh body, "" for none) as the only entry of PATH plus extra.
// It returns the exit code, the output and the file the fake node logged its cwd and arguments to. Not parallel:
// it writes an executable and runs it (the ETXTBSY note in TestRunExitCodesAndOutput).
func jsRun(t *testing.T, files map[string]string, node, extraPath string, env ...string) (int, string, string) {
	t.Helper()
	dir := t.TempDir()
	tb, bin, root := filepath.Join(dir, "toolbelt"), filepath.Join(dir, "bin"), filepath.Join(dir, "m")
	must(t, os.MkdirAll(tb, 0o755))
	must(t, os.MkdirAll(bin, 0o755))
	for _, s := range Scripts() {
		script(t, tb, s, "#!/bin/sh\nexit 0\n", 0o755)
	}
	files["A-ux/module-include.xml"] = "<x/>"
	for rel, body := range files {
		p := filepath.Join(root, rel)
		must(t, os.MkdirAll(filepath.Dir(p), 0o755))
		must(t, os.WriteFile(p, []byte(body), 0o644))
	}
	log := filepath.Join(dir, "node.log")
	if node != "" {
		script(t, bin, "node", node, 0o755)
	}
	path := bin + extraPath
	environ := append([]string{"PATH=" + path, "NODE_LOG=" + log}, env...)
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		for _, kv := range environ {
			if v, ok := strings.CutPrefix(kv, k+"="); ok {
				return v
			}
		}
		return ""
	}}
	code := mainEnv([]string{"--lint-only", root}, s, environ)
	logged, _ := os.ReadFile(log)
	return code, o.String() + e.String(), string(logged)
}

const jsLog = `echo "$PWD" > "$NODE_LOG"; echo "$@" >> "$NODE_LOG"` + "\n"

// jsRows are the js-tests rows of an output.
func jsRows(out string) []string {
	var rows []string
	for _, ln := range strings.Split(out, "\n") {
		if strings.Contains(ln, "  js-tests  ") {
			rows = append(rows, ln)
		}
	}
	return rows
}

// TestJSTestsPass (#1124): the report runs the -ux part's JS tests as `node --test <file>...` with the files
// expanded by the tool itself (a directory argument fails on Node 22), in the part's own directory, in byte order,
// *.test.js and *.test.cjs only, never under node_modules or a dot directory; a pass is one PASS row, exit 0.
func TestJSTestsPass(t *testing.T) {
	files := map[string]string{
		"A-ux/srcTest/rc/b.test.js":             "",
		"A-ux/srcTest/rc/a.test.js":             "",
		"A-ux/srcTest/rc/sub/c.test.cjs":        "",
		"A-ux/srcTest/rc/spec/ASpec.js":         "",
		"A-ux/srcTest/rc/d.test.json":           "",
		"A-ux/srcTest/node_modules/x/y.test.js": "",
		"A-ux/srcTest/.hidden/z.test.js":        "",
		"A-rt/srcTest/rc/ignored.test.js":       "",
		"A-rt/module-include.xml":               "<x/>",
	}
	code, out, logged := jsRun(t, files, "#!/bin/sh\n"+jsLog+"exit 0\n", "")
	rows := jsRows(out)
	if want := "A-ux  PASS  js-tests  node --test: 3 files passed"; len(rows) != 1 || rows[0] != want {
		t.Errorf("rows %q, want exactly %q\n%s", rows, want, out)
	}
	if code != ExitClean {
		t.Errorf("exit %d, want %d\n%s", code, ExitClean, out)
	}
	want := "--test srcTest/rc/a.test.js srcTest/rc/b.test.js srcTest/rc/sub/c.test.cjs"
	if lines := strings.Split(strings.TrimSpace(logged), "\n"); len(lines) != 2 || !strings.HasSuffix(lines[0], "/m/A-ux") || lines[1] != want {
		t.Errorf("node was run as %q, want cwd .../m/A-ux and arguments %q", logged, want)
	}
}

// TestJSTestsFail (#1124): a non-zero exit is a FAIL row carrying the exit code and the tail (last 10 lines) of
// node's output on one line; the report exits 1 and prints the ISSUES verdict.
func TestJSTestsFail(t *testing.T) {
	body := "#!/bin/sh\nfor n in 01 02 03 04 05 06 07 08 09 10 11 12; do echo \"out$n\"; done\nexit 1\n"
	code, out, _ := jsRun(t, map[string]string{"A-ux/srcTest/rc/a.test.js": ""}, body, "")
	rows := jsRows(out)
	want := "A-ux  FAIL  js-tests  node --test exited 1 (1 file): out03 | out04 | out05 | out06 | out07 | out08 | out09 | out10 | out11 | out12"
	if len(rows) != 1 || rows[0] != want {
		t.Errorf("rows %q, want exactly %q", rows, want)
	}
	if code != ExitFail || !strings.Contains(out, " · 1 FAIL · ") || !strings.Contains(out, "->  ISSUES\n") {
		t.Errorf("exit %d, want %d with 1 FAIL and ISSUES:\n%s", code, ExitFail, out)
	}
}

// TestJSTestsTimeout (#1124): a run past KIT_JS_TESTS_TIMEOUT seconds is killed and is a FAIL row naming it.
func TestJSTestsTimeout(t *testing.T) {
	code, out, _ := jsRun(t, map[string]string{"A-ux/srcTest/rc/a.test.js": ""}, "#!/bin/sh\nexec sleep 30\n",
		":/usr/bin:/bin", "KIT_JS_TESTS_TIMEOUT=1")
	rows := jsRows(out)
	if want := "A-ux  FAIL  js-tests  node --test timed out after 1s (KIT_JS_TESTS_TIMEOUT)"; len(rows) != 1 || rows[0] != want || code != ExitFail {
		t.Errorf("exit %d, rows %q, want exit 1 and %q", code, rows, want)
	}
}

// TestJSTestsNoFiles (#1124): a srcTest/ with no *.test.js / *.test.cjs is a SKIP row with its reason (node is not
// run); a -ux part with no srcTest/ at all prints no js-tests row, so a report of a module with no JS suite is as
// it was.
func TestJSTestsNoFiles(t *testing.T) {
	code, out, logged := jsRun(t, map[string]string{"A-ux/srcTest/rc/spec/ASpec.js": ""}, "#!/bin/sh\n"+jsLog+"exit 1\n", "")
	if rows, want := jsRows(out), "A-ux  SKIP  js-tests  no *.test.js or *.test.cjs file under srcTest/ (JS tests not run)"; len(rows) != 1 || rows[0] != want {
		t.Errorf("rows %q, want exactly %q", rows, want)
	}
	if code != ExitClean || logged != "" {
		t.Errorf("exit %d, node log %q: want exit 0 and node never run", code, logged)
	}
	_, out, _ = jsRun(t, map[string]string{"A-ux/src/rc/index.html": ""}, "#!/bin/sh\nexit 1\n", "")
	if rows := jsRows(out); len(rows) != 0 {
		t.Errorf("no srcTest/: rows %q, want none", rows)
	}
}

// TestJSTestsNoNode (#1124): test files and no node on PATH is a SKIP row naming the tool, never a silent pass and
// never an env fault; KIT_NODE names another node and wins over PATH.
func TestJSTestsNoNode(t *testing.T) {
	files := map[string]string{"A-ux/srcTest/rc/a.test.js": "", "A-ux/srcTest/rc/b.test.js": ""}
	code, out, _ := jsRun(t, files, "", "")
	rows := jsRows(out)
	if want := "A-ux  SKIP  js-tests  node not on PATH: 2 JS test files not run (install node >= 18 or set KIT_NODE)"; len(rows) != 1 || rows[0] != want {
		t.Errorf("rows %q, want exactly %q", rows, want)
	}
	if code != ExitClean {
		t.Errorf("exit %d, want %d (a SKIP is not a FAIL)\n%s", code, ExitClean, out)
	}
	alt := filepath.Join(t.TempDir(), "mynode")
	must(t, os.WriteFile(alt, []byte("#!/bin/sh\nexit 0\n"), 0o755))
	_, out, _ = jsRun(t, files, "", "", "KIT_NODE="+alt)
	if rows, want := jsRows(out), "A-ux  PASS  js-tests  node --test: 2 files passed"; len(rows) != 1 || rows[0] != want {
		t.Errorf("KIT_NODE: rows %q, want exactly %q", rows, want)
	}
}
