package uxversion

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the wrapper contract, written by hand from the doctrine (there is no bash lint to record):
// usage and argument errors, the empty-root ERROR row, the version lookup, stale / missing / waived rows,
// --strict, what is not a local resource, and what is never read (comments, vendor/, ext/, *.min.js, non -ux).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 18) }
