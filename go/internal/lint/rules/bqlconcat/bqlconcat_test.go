package bqlconcat

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-bql-string-concat.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 18) }

// TestUnreadable: an unreadable sub-directory and file keep the awk_lint_list contract (ContinueOnWalkError):
// both env lines, the rows of the files that were read, exit 3.
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  q = BqlQuery.make(\"x=\" + u);\n}\n")
}
