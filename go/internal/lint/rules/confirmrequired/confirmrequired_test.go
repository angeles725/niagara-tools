package confirmrequired

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule). cr-missing and cr-names hold the FAIL
// rows (no flags attribute, flags without CONFIRM_REQUIRED, each word of the heuristic); cr-flagged,
// cr-numeric and cr-multiline the flag in its forms (`|`, qualified, numeric, split over lines); cr-waiver
// the marker (same line, line above, bare, too far); cr-bare-any a bare marker on a non-destructive or a
// flagged action; cr-bare-both a bare marker beside a reasoned one; cr-nonliteral(-strict) the WARN row of a
// name that is no single string literal (exit 0, 1 under --strict); cr-forms the annotation shapes;
// cr-literals the text that is never read as code; the rest the argument contract.
//
// The 15 is the number of testdata/*.txt cases: golden.Run fails when it finds fewer, so a case deleted or
// renamed away cannot leave the rule with less proof and a green test. Add a case, raise the number.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }
