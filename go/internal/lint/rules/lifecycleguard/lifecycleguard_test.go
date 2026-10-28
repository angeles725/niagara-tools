package lifecycleguard

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. LG1 a changed/added/removed callback whose first statement
// after any super call is not an if on isRunning(); LG2 a stationStarted() with work and no started() in the same
// class (an anonymous component class too). A guarded callback, a super-only body, a wrong signature, comments and strings are
// clean; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 5) }
