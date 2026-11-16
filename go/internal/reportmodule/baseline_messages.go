package reportmodule

// The texts of the FAIL baseline (baseline.go, baseline_write.go): the file, the group headers, the rows about
// the file itself and the usage paragraph.

// baselineName is the baseline file, in the module root report-module runs on and committed with the module.
const baselineName = ".report-baseline"

// baselineCheck is the check id of the rows about the baseline file itself.
const baselineCheck = "report-baseline"

// baselineHeader opens a file written by --write-baseline ("#" lines are comments).
const baselineHeader = "# report-module baseline: the FAIL rows this module already had (report-module.sh --write-baseline).\n" +
	"# One key per line: <artifact>  <check>  <detail>. Remove a line when its FAIL is fixed; never add one by hand.\n"

const (
	msgNewRows      = "report-module: NEW FAIL rows (not in %s):\n"
	msgBaseRows     = "report-module: BASE FAIL rows (listed in %s, not part of the verdict):\n"
	msgNoRows       = "(none)\n"
	msgFailSplit    = " (%d new, %d base in %s)"
	msgStale        = "stale baseline line (no current FAIL row matches it; remove it): "
	msgFloorListed  = "floor check row cannot be baselined (it counts as NEW; remove the line): "
	msgCrashListed  = "crashed member row cannot be baselined (it counts as NEW; remove the line): "
	msgCannotRead   = "report-module: cannot read %s: %s\n"
	msgCannotWrite  = "report-module: cannot write %s: %s\n"
	msgWrote        = "report-module: wrote %s (%s)\n"
	msgFloorLeftOut = "; %s left out: verify-module and schema-risk rows are never baselined"
	msgCrashLeftOut = "; %s left out: an \"exited N with no FAIL row\" row is never baselined"
)

// baselineUsage follows the usage line.
const baselineUsage = `  baseline: <module-root>/.report-baseline lists the FAIL rows the module already had, one key per line:
      <artifact>  <check>  <detail>
    the row without its status; in <detail> a path is cut to its file name and a number standing alone (a line
    number, a count) reads N, so a key survives unrelated edits. "#" lines and blank lines are ignored.
    With the file, FAIL rows are printed in two groups, NEW and BASE, after the other rows, and the verdict and
    the exit code read the NEW rows only; a line no FAIL row matches is a WARN "stale baseline line" row.
    Never baselined (always NEW): verify-module gate rows, schema-risk rows, a member that "exited N with no
    FAIL row" (a crash), triage-console lines.
  --write-baseline  write the file from this run's FAIL rows (sorted, de-duplicated); the exit code is the run's
  --no-baseline     ignore the file for this run
`
