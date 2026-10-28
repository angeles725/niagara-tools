package charsetdefault

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. CD1 a no-argument getBytes(); CD2 a byte-array new String
// with one or three arguments (the array a declared byte[] or a getBytes/toByteArray/readAllBytes call); CD3 a
// one-argument InputStreamReader / OutputStreamWriter; CD4 any FileReader / FileWriter. A charset argument, a char[]
// or String argument, comments and string literals are clean; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 9) }
