package silentprotection

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/lint-silent-protection.bats run on a tree and every edge tree, recorded from the
// bash lint (the empty-root, unreadable and literal cases are table tests below).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 39) }

