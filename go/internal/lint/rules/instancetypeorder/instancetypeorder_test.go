package instancetypeorder

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule). ito-after holds the bad class and a good
// one side by side; ito-slotomatic reads a class with no @NiagaraType whose TYPE is a Sys.loadType field;
// ito-clean holds the classes that are never flagged; ito-literals holds INSTANCE and TYPE text in comments
// and string literals; ito-nested keeps each class's own members apart.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 7) }
