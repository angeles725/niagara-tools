package writeops_test

// n4_ack_alarm by selection: the held dry-run list and the no-argument refusal (verifier notes
// on PR #808).

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A newer dry run of the same selection that matches nothing drops the held list, so the older
// token can no longer acknowledge anything (verifier note on PR #808, probe P5c).
func TestAckSelectNothingDryRunDropsTheHeldSelection(t *testing.T) {
	db := newSelectDB()
	r := newAckRig(t, db)
	args := []any{"select", sel("source", "slot:/Services")}
	text, isErr, dry := r.call(t, args...)
	if isErr {
		t.Fatal(text)
	}
	for id := range db.state {
		db.state[id] = "acked" // acknowledged elsewhere after the first dry run
	}
	if text, isErr, again := r.call(t, args...); isErr || get(again, "confirmation_token") != nil {
		t.Fatalf("second dry run %v %s", isErr, text)
	}
	text, isErr, _ = r.call(t, append(args, "dry_run", false, "confirmation_token", get(dry, "confirmation_token"))...)
	if !isErr || !strings.Contains(text, "no dry run of this selection") || len(db.calls) != 0 {
		t.Errorf("old token after a nothing dry run: %v %s (calls %q)", isErr, text, db.calls)
	}
}

// With neither uuids nor select the tool refuses with its own error (the schema no longer
// requires uuids): a tool error, not JSON-RPC -32602 (verifier note 2 on PR #808).
func TestAckWithNeitherUuidsNorSelectIsAToolError(t *testing.T) {
	db := newSelectDB()
	reply := newAckRig(t, db).srv.Dispatch(obj("id", 1, "method", "tools/call",
		"params", obj("name", "n4_ack_alarm", "arguments", obj())))
	if e, _ := reply.Get("error"); e != nil {
		t.Fatalf("JSON-RPC error %s", pyjson.Repr(e))
	}
	if got := pyjson.Repr(reply); !strings.Contains(got, "give uuids or select") || len(db.ords) != 0 {
		t.Errorf("reply %s", got)
	}
}
