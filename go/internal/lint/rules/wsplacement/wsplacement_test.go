package wsplacement

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-ws-placement.bats run (WSP-awkfail and WSP-finderr aside: TestUnreadable pins
// them) and every edge tree, recorded from the bash lint-ws-placement.sh.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 39) }

// TestUnreadable: WSP-awkfail and WSP-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  void a() {\n    add(\"n\", new BFolder());\n  }\n}\n")
}
