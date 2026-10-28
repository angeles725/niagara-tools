package permissionsxml

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G21 contract (ledger.d/2026-10-23-r8-p1-sign.md), one hand-written case per testdata/*.txt: the
// kit template is clean; a root other than <permissions> and a group type other than station|workbench|all FAIL; a
// req-permission or opt-permission without <name> or <purposeKey> WARNs; new File("/…") and Paths.get("/…") in -rt
// src/ WARN while new File(Sys.getStationHome(), …), a comment, a string and srcTest/ are clean; --strict; a
// malformed file is exit 3; empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 10) }
