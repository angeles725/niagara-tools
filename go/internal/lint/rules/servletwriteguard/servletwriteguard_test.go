package servletwriteguard

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the cases of testdata/, written by hand from the rule (truth cases, no bash original).
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 9) }
