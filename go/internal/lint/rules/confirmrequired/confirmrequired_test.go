package confirmrequired

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule). cr-missing and cr-names hold the FAIL
// rows (no flags attribute, flags without CONFIRM_REQUIRED, each word of the heuristic); cr-flagged,
// cr-numeric and cr-multiline the flag in its forms (`|`, qualified, numeric, split over lines); cr-waiver
// the marker (same line, line above, bare, too far); cr-forms the annotation shapes; cr-literals the text
// that is never read as code; the rest the argument contract.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 11) }
