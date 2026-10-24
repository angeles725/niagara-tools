package inertcoordination

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-inert-coordination.bats run (ICO-awkfail and ICO-finderr aside: TestUnreadable
// pins them) and every edge tree, recorded from the bash lint-inert-coordination.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 28) }

// TestUnreadable: ICO-awkfail and ICO-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  long staggerDelay;\n  List<B> units() {\n    return Collections.singletonList(a);\n  }\n}\n")
}
