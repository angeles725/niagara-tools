package reportmodule

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Two FAIL rows of ordinary lints (the second lint has two rows that differ only by their line number).
const (
	stubDelays = "printf 'FAIL  lint-delays  /w/A-rt/src/A.java:12  sleep of 3000 ms in a loop\\n'; exit 1\n"
	stubSysOut = "printf 'FAIL  lint-no-system-out  /w/B.java:7  System.out\\nFAIL  lint-no-system-out  /w/B.java:9  System.out\\n'; exit 1\n"
	keyDelays  = "A-rt  lint-delays  A.java:N  sleep of N ms in a loop"
	keySysOut  = "A-rt  lint-no-system-out  B.java:N  System.out"
	rowDelays  = "A-rt  FAIL  lint-delays  A.java:12  sleep of 3000 ms in a loop\n"
	rowsSysOut = "A-rt  FAIL  lint-no-system-out  B.java:7  System.out\nA-rt  FAIL  lint-no-system-out  B.java:9  System.out\n"
)

var twoLints = map[string]string{"lint-delays": stubDelays, "lint-no-system-out": stubSysOut}

// baselineModule is a one-artifact module whose floor checks run: a built jar and a .deploy-baseline directory.
func baselineModule(t *testing.T) string {
	t.Helper()
	return tree(t, "A-rt/src/", "A-rt/build/libs/a.jar", "A-rt/.deploy-baseline/")
}

// baselineRun runs report-module on args through mainEnv with a toolbelt of stubs: every member prints nothing
// and exits 0 but the ones in stubs (script name -> bash body).
func baselineRun(t *testing.T, stubs map[string]string, args ...string) (code int, out, errs string) {
	t.Helper()
	tb := t.TempDir()
	for _, s := range Scripts() {
		body := "exit 0\n"
		if b, ok := stubs[strings.TrimSuffix(s, ".sh")]; ok {
			body = b
		}
		script(t, tb, s, "#!/usr/bin/env bash\n"+body, 0o755)
	}
	var o, e bytes.Buffer
	s := cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == ToolbeltEnv {
			return tb
		}
		return ""
	}}
	code = mainEnv(args, s, []string{"PATH=/usr/bin:/bin"})
	return code, o.String(), e.String()
}

func writeBaselineFile(t *testing.T, root string, lines ...string) string {
	t.Helper()
	p := filepath.Join(root, baselineName)
	must(t, os.WriteFile(p, []byte(strings.Join(lines, "\n")+"\n"), 0o644))
	return p
}

// summaryLine is the last line of a report.
func summaryLine(out string) string {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	return lines[len(lines)-1]
}

// Decision 6: with no .report-baseline and no baseline option the report is the one it always was: FAIL rows in
// place, the plain FAIL count, nothing on stderr, no file written.
func TestNoBaselineFileLeavesTheReportAsItWas(t *testing.T) {
	root := baselineModule(t)
	code, out, errs := baselineRun(t, twoLints, root)
	want := "report-module: 1 artifact · 19 PASS · 3 FAIL · 0 WARN · 2 SKIP · 0 ADVISORY  ->  ISSUES"
	if code != ExitFail || errs != "" || summaryLine(out) != want || !strings.Contains(out, rowDelays+"A-rt  PASS  lint-link-target-flags") ||
		!strings.Contains(out, rowsSysOut) || strings.Contains(out, "NEW") || strings.Contains(out, "BASE") || strings.Contains(out, "baseline") {
		t.Errorf("exit %d, stderr %q, want exit 1, the rows in place and %q:\n%s", code, errs, want, out)
	}
	if _, err := os.Lstat(filepath.Join(root, baselineName)); err == nil {
		t.Error("a run without --write-baseline wrote " + baselineName)
	}
}

// Decision 2: --write-baseline writes the run's FAIL rows as sorted, de-duplicated keys and leaves stdout and
// the exit code as they were.
func TestWriteBaselineWritesTheFailKeys(t *testing.T) {
	root := baselineModule(t)
	plainCode, plainOut, _ := baselineRun(t, twoLints, root)
	code, out, errs := baselineRun(t, twoLints, root, "--write-baseline")
	path := root + "/" + baselineName
	if code != plainCode || code != ExitFail || out != plainOut || errs != "report-module: wrote "+path+" (2 FAIL keys)\n" {
		t.Errorf("exit %d (plain %d), stderr %q; stdout same as the plain run: %v", code, plainCode, errs, out == plainOut)
	}
	got, err := os.ReadFile(path)
	must(t, err)
	if want := baselineHeader + keyDelays + "\n" + keySysOut + "\n"; string(got) != want {
		t.Errorf("%s =\n%s\nwant\n%s", baselineName, got, want)
	}
	if code, out, _ := baselineRun(t, twoLints, root); code != ExitClean || !strings.HasSuffix(summaryLine(out), "->  CLEAN") {
		t.Errorf("the run after --write-baseline: exit %d, %q; want 0 CLEAN", code, summaryLine(out))
	}
}

// Decision 3: with a baseline the FAIL rows are grouped NEW and BASE, the summary states both counts and names
// the file, and the verdict and the exit code read the NEW rows only.
func TestBaselineGroupsNewAndBaseRows(t *testing.T) {
	root := baselineModule(t)
	path := writeBaselineFile(t, root, "# known", "", keyDelays)
	code, out, errs := baselineRun(t, twoLints, root)
	groups := "(run)  SKIP  triage-console  no --console-dir\n" +
		"report-module: NEW FAIL rows (not in " + path + "):\n" + rowsSysOut +
		"report-module: BASE FAIL rows (listed in " + path + ", not part of the verdict):\n" + rowDelays +
		"report-module: 1 artifact · 19 PASS · 3 FAIL (2 new, 1 base in " + path + ") · 0 WARN · 2 SKIP · 0 ADVISORY  ->  ISSUES\n"
	if code != ExitFail || errs != "" || !strings.HasSuffix(out, groups) || strings.Count(out, "  FAIL  ") != 3 {
		t.Errorf("exit %d, stderr %q, want exit 1 and the report to end with\n%s\ngot\n%s", code, errs, groups, out)
	}
	writeBaselineFile(t, root, keySysOut, keyDelays)
	code, out, errs = baselineRun(t, twoLints, root)
	groups = "report-module: NEW FAIL rows (not in " + path + "):\n(none)\n" +
		"report-module: BASE FAIL rows (listed in " + path + ", not part of the verdict):\n" + rowDelays + rowsSysOut +
		"report-module: 1 artifact · 19 PASS · 3 FAIL (0 new, 3 base in " + path + ") · 0 WARN · 2 SKIP · 0 ADVISORY  ->  CLEAN\n"
	if code != ExitClean || errs != "" || !strings.HasSuffix(out, groups) {
		t.Errorf("exit %d, stderr %q, want exit 0 and the report to end with\n%s\ngot\n%s", code, errs, groups, out)
	}
}

// Decision 5: a baseline line no current FAIL row matches is a WARN row and changes no exit code.
func TestStaleBaselineLineIsAWarnRow(t *testing.T) {
	root := baselineModule(t)
	path := writeBaselineFile(t, root, "A-rt  lint-gone  Old.java:N  fixed long ago", keyDelays)
	code, out, errs := baselineRun(t, map[string]string{"lint-delays": stubDelays}, root)
	want := rowDelays +
		"(module)  WARN  report-baseline  stale baseline line (no current FAIL row matches it; remove it): A-rt  lint-gone  Old.java:N  fixed long ago\n" +
		"report-module: 1 artifact · 20 PASS · 1 FAIL (0 new, 1 base in " + path + ") · 1 WARN · 2 SKIP · 0 ADVISORY  ->  CLEAN\n"
	if code != ExitClean || errs != "" || !strings.HasSuffix(out, want) {
		t.Errorf("exit %d, stderr %q, want exit 0 and the report to end with\n%s\ngot\n%s", code, errs, want, out)
	}
}

// Decision 7: --no-baseline reads the report as if the file were not there.
func TestNoBaselineIgnoresTheFile(t *testing.T) {
	root := baselineModule(t)
	wantCode, wantOut, wantErr := baselineRun(t, twoLints, root)
	path := writeBaselineFile(t, root, keyDelays, keySysOut)
	code, out, errs := baselineRun(t, twoLints, root, "--no-baseline")
	if code != wantCode || code != ExitFail || out != wantOut || errs != wantErr {
		t.Errorf("--no-baseline: exit %d, stderr %q, stdout\n%s\nwant the report of a module with no baseline (exit %d):\n%s", code, errs, out, wantCode, wantOut)
	}
	if got, _ := os.ReadFile(path); string(got) != keyDelays+"\n"+keySysOut+"\n" {
		t.Errorf("--no-baseline changed the file:\n%s", got)
	}
}

// A baseline that cannot be read is an env fault before any member runs, never a run without it.
func TestUnreadableBaselineIsAnEnvFault(t *testing.T) {
	root := baselineModule(t)
	must(t, os.Mkdir(filepath.Join(root, baselineName), 0o755))
	code, out, errs := baselineRun(t, nil, root)
	if code != ExitEnv || out != "" || !strings.HasPrefix(errs, "report-module: cannot read "+root+"/"+baselineName+": ") {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 3 and no row", code, out, errs)
	}
}

func TestRowKeyIsStable(t *testing.T) {
	t.Parallel()
	tests := []struct{ detail, want string }{
		{"A.java:12  sleep of 3000 ms in a loop", "L  c  A.java:N  sleep of N ms in a loop"},
		{"3", "L  c  N"},
		{"clean (wiring map: /home/u/m/../docs/wiring-map.md)", "L  c  clean (wiring map: wiring-map.md)"},
		{"app.js:4:17  needs ES2020 (Chrome 80), hmi1Write at src/rc/", "L  c  app.js:N:N  needs ES2020 (Chrome N), hmi1Write at rc"},
		{"", "L  c"},
	}
	for _, tt := range tests {
		if got := rowKey(failRow{"L", "c", tt.detail}); got != tt.want {
			t.Errorf("rowKey(%q) = %q, want %q", tt.detail, got, tt.want)
		}
	}
}
