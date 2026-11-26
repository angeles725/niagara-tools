package dashboardsave

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the cases of testdata/, written by hand from the rule (retro
// lint-dashboard-write-save Δ1, issue #821; no bash original) and the per-path and stop-flush cases (issue #1118).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 13) }
