package reportmodule

import (
	"os"
	"strings"
	"testing"
)

// The rows a baseline can never cover: floor checks and crashed members.

// Decision 4: a verify-module gate row and a schema-risk row are NEW even when the baseline lists them, and
// --write-baseline leaves them out and says so.
func TestFloorRowsAreNeverBaselined(t *testing.T) {
	root := baselineModule(t)
	stubs := map[string]string{
		"lint-delays":   stubDelays,
		"verify-module": "printf 'FAIL signature a.jar unsigned\\n'; exit 1\n",
		"schema-risk":   "exit 2\n",
	}
	keySig, keySchema := "A-rt  signature  unsigned", "A-rt  schema-risk  verdict=OUTAGE"
	path := writeBaselineFile(t, root, keyDelays, keySig, keySchema)
	code, out, _ := baselineRun(t, stubs, root)
	want := "report-module: NEW FAIL rows (not in " + path + "):\n" +
		"A-rt  FAIL  signature  unsigned\nA-rt  FAIL  schema-risk  verdict=OUTAGE\n" +
		"report-module: BASE FAIL rows (listed in " + path + ", not part of the verdict):\n" + rowDelays +
		"(module)  WARN  report-baseline  floor check row cannot be baselined (it counts as NEW; remove the line): " + keySig + "\n" +
		"(module)  WARN  report-baseline  floor check row cannot be baselined (it counts as NEW; remove the line): " + keySchema + "\n" +
		"report-module: 1 artifact · 18 PASS · 3 FAIL (2 new, 1 base in " + path + ") · 2 WARN · 2 SKIP · 0 ADVISORY  ->  ISSUES\n"
	if code != ExitFail || !strings.HasSuffix(out, want) {
		t.Errorf("exit %d, want exit 1 and the report to end with\n%s\ngot\n%s", code, want, out)
	}
	code, _, errs := baselineRun(t, stubs, root, "--write-baseline")
	got, err := os.ReadFile(path)
	must(t, err)
	wantErr := "report-module: wrote " + path + " (1 FAIL key; 2 floor check rows left out: verify-module and schema-risk rows are never baselined)\n"
	if code != ExitFail || errs != wantErr || string(got) != baselineHeader+keyDelays+"\n" {
		t.Errorf("--write-baseline: exit %d, stderr %q, file\n%s", code, errs, got)
	}
}

// Decision 4, second half: a SKIPped floor check reads INCOMPLETE (exit 4) with every FAIL row baselined.
func TestBaselineKeepsTheIncompleteVerdict(t *testing.T) {
	root := tree(t, "A-rt/src/") // no built jar, no .deploy-baseline
	path := writeBaselineFile(t, root, keyDelays)
	code, out, _ := baselineRun(t, map[string]string{"lint-delays": stubDelays}, root)
	want := "report-module: 1 artifact · 18 PASS · 1 FAIL (0 new, 1 base in " + path + ") · 0 WARN · 4 SKIP · 0 ADVISORY  ->  INCOMPLETE"
	if code != ExitIncomplete || summaryLine(out) != want {
		t.Errorf("exit %d, %q; want exit 4 and %q", code, summaryLine(out), want)
	}
}

// A member that exited non-zero without a FAIL row (a crash) is NEW even when the baseline lists its row, for
// any member, and --write-baseline leaves the row out and says so.
func TestCrashedMemberRowIsNeverBaselined(t *testing.T) {
	root := baselineModule(t)
	stubs := map[string]string{"lint-delays": "exit 1\n", "lint-no-system-out": stubSysOut}
	crash := "exited 1 with no FAIL row (crash or unexpected output) -- run lint-delays.sh manually"
	keyCrash := "A-rt  lint-delays  exited N with no FAIL row (crash or unexpected output) -- run lint-delays.sh manually"
	path := writeBaselineFile(t, root, keyCrash, keySysOut)
	code, out, _ := baselineRun(t, stubs, root)
	want := "report-module: NEW FAIL rows (not in " + path + "):\nA-rt  FAIL  lint-delays  " + crash + "\n" +
		"report-module: BASE FAIL rows (listed in " + path + ", not part of the verdict):\n" + rowsSysOut +
		"(module)  WARN  report-baseline  crashed member row cannot be baselined (it counts as NEW; remove the line): " + keyCrash + "\n" +
		"report-module: 1 artifact · 18 PASS · 3 FAIL (1 new, 2 base in " + path + ") · 1 WARN · 2 SKIP · 0 ADVISORY  ->  ISSUES\n"
	if code != ExitFail || !strings.HasSuffix(out, want) {
		t.Errorf("exit %d, want exit 1 and the report to end with\n%s\ngot\n%s", code, want, out)
	}
	code, _, errs := baselineRun(t, stubs, root, "--write-baseline")
	got, err := os.ReadFile(path)
	must(t, err)
	wantErr := "report-module: wrote " + path + " (1 FAIL key; 1 crashed member row left out: an \"exited N with no FAIL row\" row is never baselined)\n"
	if code != ExitFail || errs != wantErr || string(got) != baselineHeader+keySysOut+"\n" {
		t.Errorf("--write-baseline: exit %d, stderr %q, file\n%s", code, errs, got)
	}
}
