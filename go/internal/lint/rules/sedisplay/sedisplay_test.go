package sedisplay

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-se-display.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 20) }

// TestUnreadable: an unreadable sub-directory and file keep the awk_lint_list contract (ContinueOnWalkError):
// both env lines, the rows of the files that were read, exit 3.
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "import javax.swing.JFrame;\nclass A {}\n")
}
