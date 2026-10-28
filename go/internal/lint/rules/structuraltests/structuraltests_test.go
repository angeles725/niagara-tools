package structuraltests

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G34 contract (ledger.d/2026-10-23-r8-p1-tests.md), one hand-written case per testdata/*.txt: a
// srcTest file reading a .java source as text WARNs when its part's srcTest has no BTestNg/BTestNgStation class
// (simple or qualified name) and is clean when it has one; a .json fixture read, a .java name in a comment and
// a part with no reader are clean; parts are judged on their own; --strict prints FAIL; empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 8) }
