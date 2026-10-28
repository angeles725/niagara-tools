package slotdefaulttype

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. A defaultValue literal whose Java type the slot's type does not
// accept FAILs (int with false or 2d, boolean with 0, double with BBoolean.TRUE, float with 1.5, String with 0); an int
// literal on a long/double slot is clean (the introspector widens it), and so are a matching literal, a matching
// B-value (BInteger.make, BString.make) and an expression the rule cannot type (a constant, new BStatusNumeric());
// comments and multi-line annotations; empty root; usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }
