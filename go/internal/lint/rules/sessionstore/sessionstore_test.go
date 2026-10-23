package sessionstore

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-session-store-lazy-evict.bats run and every edge tree, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 38) }

// TestUnreadable: SSL-awkfail and SSL-finderr keep awk_lint_run's contract (ContinueOnWalkError).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  static Map<String, E> S = null;\n  void a(String k) {\n    S.put(k, null);\n  }\n}\n")
}
