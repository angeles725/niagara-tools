package servlet

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-servlet.bats run and every edge tree, recorded from the bash lint. A case
// whose text starts with a note before "-- args --" is a declared difference from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 45) }

// TestUnreadable: the bash find printed its error and the loop scanned every file it listed, so the
// spec sets ContinueOnWalkError (the engine adds the env lines and exit 3).
func TestUnreadable(t *testing.T) {
	golden.Unreadable(t, Spec(), "class A extends BWebServlet {\n  void doPost(WebOp op) { c.set(\"a\", 1); }\n}\n")
}
