package extordvalue

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-wb-external-ord-value.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 16) }

// TestUnreadable: WEO-awkfail and WEO-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  void a() {\n    Object o = ord.get(s);\n    BStatusValue v;\n  }\n}\n")
}
