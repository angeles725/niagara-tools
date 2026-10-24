package changedhotwrite

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-changed-hot-write.bats run (CHW-awkfail and CHW-finderr aside: TestUnreadable
// pins them) and every edge tree, recorded from the bash lint-changed-hot-write.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 27) }

// TestUnreadable: CHW-awkfail and CHW-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  static final Property xHours = newProperty(0, 0.0, null);\n"+
		"  public void changed(Property p, Context cx) {\n    if (p == a) execute();\n    if (p == b) execute();\n    if (p == c) execute();\n    if (p == d) execute();\n    if (p == e) execute();\n    if (p == f) execute();\n    if (p == g) execute();\n  }\n  void execute() { setXHours(1.0); }\n}\n")
}
