package absoluteclaims

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G38 contract (ledger.d/2026-10-23-r8-p1-process.md), one hand-written case per testdata/*.txt: a
// line holding never/always/silently/forbidden/throws whose claim unit (its bullet or paragraph) cites only retro or
// corpus tags is an INFO row; a code/course/devguide/CERT/javap tag or a `.java:` line cite anywhere in the unit
// clears it; a line with no absolute word is not listed; the report never fails (--strict too); empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 5) }
