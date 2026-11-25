package asyncblock

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule), one per testdata/*.txt. blocking holds the
// G28 WARN cases (ASYNC doSubmit with Thread.sleep and no post override; HTTP, stream and BQL calls; combined
// flags); course the two course acceptance cases (the exam BHistoryExportService and Lab 9 BEngineThreadExecutor:
// WARN, exit 0); post the post() overrides (this::doX, a lambda to doX: WARN; new Invocation: clean); clean what
// is never flagged (a post override beside a sleeping handler, a non-ASYNC action, an unrelated sleep, comments
// and strings); strict, usage and empty-root the argument contract.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 8) }
