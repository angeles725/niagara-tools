package licensegate

import (
	"bytes"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-license-isoperational-gate.bats run on a readable tree and every edge tree,
// recorded from the bash lint (origin/main 89390e5).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 39) }

// TestEmptyDir: an empty root directory (the golden format cannot hold one; recorded from the bash lint as
// edge-empty, same row as edge-txtonly) is the K20 ERROR row and exit 3.
func TestEmptyDir(t *testing.T) {
	dir := t.TempDir()
	var o, e bytes.Buffer
	code := lint.Command(Spec())([]string{ID, dir}, cli.Stdio{Out: &o, Err: &e})
	if want := "ERROR  " + ID + "  " + dir + "  no files to scan (K20: wrong path or empty scaffold)\n"; code != 3 || o.String() != want || e.Len() != 0 {
		t.Errorf("exit %d out %q err %q, want 3 and %q", code, o.String(), e.String(), want)
	}
}

// TestUnreadable: LIG-awkfail and LIG-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A extends BAbstractService {\n  public Feature getLicenseFeature() {\n    return f;\n  }\n  public void doPoll() {\n    poll();\n  }\n}\n")
}
