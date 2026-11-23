package benchcall

// Before a write executes, the reply to its dry run must show dry_run: true: the proof that what
// ran first was a dry run (retro R15 delta M17).

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestADryRunReplyThatDoesNotShowDryRunTrueIsNoProof(t *testing.T) {
	plan := func(kv ...any) *pyjson.Object {
		o := pyjson.NewObject()
		for i := 0; i < len(kv); i += 2 {
			o.Set(kv[i].(string), kv[i+1])
		}
		return o
	}
	const want = "the dry run's reply does not show dry_run: true, so it is not proof of a dry run: not executing"
	for name, reply := range map[string]any{
		"executed":   plan("dry_run", false, "confirmation_token", "tok"),
		"no field":   plan("confirmation_token", "tok"),
		"not object": "text",
		"nil":        nil,
	} {
		if err := dryRunProved(reply); err == nil || err.Error() != want {
			t.Errorf("%s: err = %v, want %q", name, err, want)
		}
	}
	if err := dryRunProved(plan("dry_run", true, "confirmation_token", "tok")); err != nil {
		t.Errorf("a dry_run: true reply was refused: %v", err)
	}
}
