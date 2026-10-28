package loggername

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G40 contract (ledger.d/2026-10-23-r8-p1-process.md), one hand-written case per testdata/*.txt: with
// module AcmePan, getLogger("AcmePan") and getLogger("AcmePan.defrost") are clean, getLogger("acme")
// and getLogger(BFoo.class.getName()) WARN; the name comes from the part gradle, else the group gradle, else the part
// directory; a non-literal argument, a comment and srcTest/ are clean; --strict; empty roots and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 7) }
