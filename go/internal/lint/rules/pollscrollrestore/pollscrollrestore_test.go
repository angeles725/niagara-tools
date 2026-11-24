package pollscrollrestore

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt (#909, retro R9 ΔB1). A poll callback that clears a
// container, directly or through a function it calls, in a file with a scroller (CSS or style.overflow) and no
// scroll position code WARNs at the clear; a restore, no scroller, a clear outside a poll, a restore that is only
// a comment or a string, and a vendor/dist tree are clean; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 11) }
