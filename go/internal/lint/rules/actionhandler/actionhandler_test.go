package actionhandler

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule), one per testdata/*.txt. missing and
// signature hold the FAIL rows (a misspelled, mis-typed or non-public do method; a static one; a return type that
// differs from the action's); clean holds the handlers that are fine (Context, parameter, abstract, inherited in
// the same file, qualified return, a post override); pair the G1 cases (doX() beside doX(Context)); methodref the
// F3 cases (this::doX, new Thread(() -> doX(), outside post only); literals comments, strings and calls; strict
// the exit code of a WARN under --strict; usage and empty-root the argument contract.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 9) }
