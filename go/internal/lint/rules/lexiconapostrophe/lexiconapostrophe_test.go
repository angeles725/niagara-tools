package lexiconapostrophe

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. LX1 a lexicon value with a MessageFormat argument ({0},
// {1,number}) and a single apostrophe; a doubled pair is clean, an apostrophe in a value without arguments is clean (read with
// getText(key) it is not formatted), comments, a continuation line and a key-only line; --strict; a root with no
// lexicon is clean; usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 4) }
