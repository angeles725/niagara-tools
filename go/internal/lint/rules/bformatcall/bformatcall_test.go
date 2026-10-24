package bformatcall

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-bformat-call.bats run on a tree the format can hold and every edge tree, recorded
// from the bash lint (origin/main 45ddb3b5) on copies of the root.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 23) }

// TestUnreadable: the awk_lint_run contract (a walk error and an unreadable file, rows still printed, exit 3).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A { String f = \"%user%\"; }\n")
}
