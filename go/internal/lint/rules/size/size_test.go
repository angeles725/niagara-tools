package size

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-size.bats run (LSZ-awkfail and LSZ-finderr aside: TestUnreadable pins them)
// and every edge tree, recorded from the bash lint-size.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 43) }

// TestUnreadable: LSZ-awkfail and LSZ-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  void a() {\n"+strings.Repeat("    x();\n", 81)+"  }\n}\n")
}
