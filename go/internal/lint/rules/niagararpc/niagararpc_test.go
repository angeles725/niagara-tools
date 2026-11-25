package niagararpc

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: hand-written cases (no bash lint precedes this rule; tests/fixtures/lint-niagara-rpc holds the same
// sources for tests/lint-niagara-rpc.bats). nrpc-unrestricted: the three spellings of "no permission";
// nrpc-unguarded: set, invoke, doXxx and a typed setter; nrpc-guarded and nrpc-readonly: the clean shapes;
// nrpc-guard-order: a check after the write; nrpc-comments: comment and literal text is no code; nrpc-waiver and
// nrpc-waiver-bare: the reviewed marker; nrpc-multi, nrpc-nonliteral and nrpc-edges: the reading limits.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 18) }
