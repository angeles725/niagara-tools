package agentonincludesync

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: the rule's contract, one hand-written case per testdata/*.txt (a native rule has no bash lint
// to record): single string, array, two @AgentOn, @AgentOn in a comment or a string, a line with no <agent>,
// a partial set, a class absent from module-include.xml, a nested type, a part with no module-include.xml,
// a root with none (exit 3), a malformed file (exit 3), srcTest/ and non-src files skipped, usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 19) }
