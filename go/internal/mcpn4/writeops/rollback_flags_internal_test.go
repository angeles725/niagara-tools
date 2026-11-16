package writeops

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A journaled set-flags inverse that lacks its slot name or its flag string is never skipped: the
// rollback is refused before anything is read or sent.
func TestAMalformedSetFlagsInverseRefusesTheRollback(t *testing.T) {
	for name, op := range map[string]*pyjson.Object{
		"no slot name":   inverseOp("f", "h"),
		"no flag string": inverseOp("n", "note"),
		"flags a number": inverseOp("n", "note", "f", 1),
	} {
		err := flagsUnchangedSince(nil, nil, []any{op}, nil)
		if err == nil || !strings.Contains(err.Error(), "malformed set-flags inverse") {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

// inverseOp is a journaled set-flags op on handle 42 holding the given keys.
func inverseOp(kv ...any) *pyjson.Object {
	op := pyjson.NewObject()
	op.Set("nm", "f")
	op.Set("h", "42")
	for i := 0; i < len(kv); i += 2 {
		op.Set(kv[i].(string), kv[i+1])
	}
	return op
}
