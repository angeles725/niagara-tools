package ndrivertaglesscorrelation

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the G67 contract (ledger.d/2026-10-23-r8-u5-c59-c62.md), one hand-written case per testdata/*.txt:
// NMessage subclasses with no getTag() WARN once on the first one when no comm config returns 1 (the hint
// commented out, an override returning 4, a 1 outside a comm config); a getTag() override (also in a module-local
// message base) or a comm config, direct or through a local base, returning 1 is clean; no NMessage subclass is
// clean; a commented-out or quoted getTag() and a getTag() of an unrelated class do not count; --strict; only
// -rt src/ is read; usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 12) }
