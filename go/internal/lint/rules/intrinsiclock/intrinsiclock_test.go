package intrinsiclock

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. IL1 a synchronized instance method and IL2 a
// synchronized (this) block in a Niagara B-type class (its own name or its superclass name is B + upper case); IL3 a
// synchronized block nested in another synchronized block or method, in any class. <Outer>.this inside an anonymous
// class is IL2; a lambda body or anonymous method inside a lock is not under it (no IL3), but a lock nested inside
// that body is. A private final lock object, a static synchronized method, a non-B helper, an anonymous class's plain
// this, comments and strings are clean; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }
