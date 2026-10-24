package subscribeleak

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-subscribe-without-unsubscribe.bats run and every edge tree, recorded from the bash
// lint, plus the hand-written truth-* cases of the comment- and string-aware match.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 21) }

// TestUnreadable: an unreadable sub-directory and file keep the awk_lint_list contract (ContinueOnWalkError):
// both env lines, the rows of the files that were read, exit 3.
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A {\n  void f() { s.subscribe(c); }\n}\n")
}
