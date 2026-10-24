package machinedefaultpath

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the wrapper contract, written by hand from the doctrine (there is no bash lint to record):
// usage, the empty-root ERROR row, srcTest/ and *-worktrees/ pruned, --strict and several roots.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 5) }
