package extwritable

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: every tests/ext-writable-shape.bats run on a tree and every edge tree, recorded from the bash
// lint (the empty-root and unreadable cases are table tests).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 40) }
