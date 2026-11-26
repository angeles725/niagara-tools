package propertyaccessors

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. A B* class with a hand-declared newProperty slot and a
// getter but no setter FAILs (PA1, the issue #1159 reproduction); a non-public setter or a missing getter FAILs
// (PA2); the generated-style class (get/set, an is<X> getter for a boolean) is clean; a class that is not B+upper,
// a non-static or non-newProperty field, and a slot of a nested class (judged against the nested class's own
// methods, never the outer's, and not judged at all) are not judged; comments and strings are not code; usage and
// the empty root.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 5) }
