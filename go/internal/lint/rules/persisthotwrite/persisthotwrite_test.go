package persisthotwrite

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-persist-hot-write.bats run (PHW-awkfail and PHW-finderr aside: TestUnreadable
// pins them) and every edge tree, recorded from the bash lint-persist-hot-write.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 34) }

// TestUnreadable: PHW-awkfail and PHW-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  static final Property xHours = newProperty(0, 0.0, null);\n"+
		"  public void changed(Property p, Context cx) {\n    setXHours(getXHours() + 1.0);\n  }\n}\n")
}
