package setnullord

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-set-null-ord.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 32) }

// TestUnreadable: SNO-awkfail and SNO-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  void a(BComponent p, M m) {\n    m.setX(p.getSlotPathOrd());\n  }\n}\n")
}
