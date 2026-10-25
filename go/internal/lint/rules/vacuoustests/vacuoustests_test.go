package vacuoustests

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the rule's contract, one hand-written case per testdata/*.txt (a native rule has no bash lint to
// record): the three FAIL shapes (trivially true, no assertion, always skipped), a helper-asserted test (same
// class, srcTest/ base, enclosing class, a static import of a srcTest/ class) and expected exceptions that PASS,
// the one-level helper depth, a @BeforeMethod and other non-@Test methods not counted, a named allowance (PASS row
// with its reason), a stale allowance (FAIL), --allow without a reason or a Class.method name, an unparseable
// source and an empty root (exit 3), and the usage line.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }
