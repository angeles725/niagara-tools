package postpermission

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/golden"
)

// TestGolden: one hand-written case per testdata/*.txt. A request or RPC handler (doGet/doPost/doPut/doDelete/
// service, a method taking a WebOp or a servlet request/response, @NiagaraRpc) or a do<Action> handler taking a
// Context that calls post( with no permission check before it in the same method FAILs; every accepted guard
// form, a waiver comment with a reason, an internal timer post (no Context, no request) and a post( that is only
// a comment or a string are clean; --strict; the empty root and usage.
func TestGolden(t *testing.T) { golden.Run(t, Spec(), 15) }
