package slotomaticmarkers

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule). sm-zero is the BHmiStatus shape
// (Cliente/panccadia-leon DashboardPan-rt, region with no stamp, commit de36722) beside a sibling with one
// stamp and the Slot-o-Matic "Generated" line; sm-literal holds marker and stamp text inside literals.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 13) }
