package intrinsiclock

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. IL1 a synchronized instance method and IL2 a
// synchronized (this) block in a Niagara B-type class (its own name or its superclass name is B + upper case); IL3 a
// synchronized block nested in another synchronized block or method, in any class. A private final lock object, a
// static synchronized method, a non-B helper, an anonymous class's this, comments and strings are clean; --strict;
// the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }
