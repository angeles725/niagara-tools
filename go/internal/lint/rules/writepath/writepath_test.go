package writepath

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-write-path.bats and tests/write-path-drift.bats run, the report-module
// fixtures and the edge trees, recorded from the bash lint.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 65) }
