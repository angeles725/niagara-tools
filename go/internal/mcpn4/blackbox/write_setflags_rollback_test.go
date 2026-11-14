package blackbox

// n4_rollback of an n4_set_flags batch: the journaled inverse is a set-flags op, replayed and
// verified by read-back.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestRollbackOfSetFlagsRestoresThePreviousFlagsAndVerifiesThem(t *testing.T) {
	for _, previous := range []string{"h", ""} { // "": the restored slot is listed without flags again
		f := newFlagsRig(t, previous)
		out := f.Srv.RunWrite("n4_set_flags", f.args("os"))
		reply := f.Srv.Dry("n4_rollback", Args{"batch_id": out["batch_id"]})
		if ops := planOf(reply)["ops"]; !same(ops, []any{f.op(previous)}) {
			t.Errorf("previous %q: rollback ops = %v", previous, ops)
		}
		if got := f.flags(); got != "so" { // the dry run restored nothing
			t.Errorf("previous %q: flags after the dry run = %q", previous, got)
		}
		back := f.Srv.OK("n4_rollback", Execute(Args{"batch_id": out["batch_id"]}, reply["confirmation_token"]))
		if back["verdict"] != "verified" || !same(back["observed"], map[string]any{"restored": true}) ||
			!same(back["requested"], map[string]any{"rolled_back": out["batch_id"]}) {
			t.Errorf("previous %q: back = %v", previous, back)
		}
		if got := f.flags(); got != previous {
			t.Errorf("previous %q: flags = %q", previous, got)
		}
		if got := f.sent(); !same(got, []any{f.op("so"), f.op(previous)}) {
			t.Errorf("previous %q: sent = %v", previous, got)
		}
		if view := batchView(t, f.Rig, back["batch_id"]); view["rollback_of"] != out["batch_id"] || view["tool"] != "n4_rollback" ||
			view["verdict"] != "verified" {
			t.Errorf("previous %q: journal view = %v", previous, view)
		}
	}
}

// Compare before restore: flags someone changed after the batch are not overwritten.
func TestRollbackOfSetFlagsIsRefusedWhenTheFlagsChangedSince(t *testing.T) {
	f := newFlagsRig(t, "h")
	out := f.Srv.RunWrite("n4_set_flags", f.args("ho"))
	f.setFlags("r")
	text := f.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	for _, needle := range []string{"the flags of slot 'note' on " + f.ord + " changed since batch", `the batch set "ho" (h hidden, o operator)`,
		`the station now holds "r" (r readonly)`, "n4_set_flags"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if got := f.flags(); got != "r" || len(f.sent()) != 1 {
		t.Errorf("flags %q, sent %v", got, f.sent())
	}
}

// The rollback's own verdict comes from the read-back too: flags the station did not restore
// are a mismatch.
func TestRollbackOfSetFlagsThatTheStationDoesNotRestoreIsAMismatch(t *testing.T) {
	f := newFlagsRig(t, "h")
	out := f.Srv.RunWrite("n4_set_flags", f.args("ho"))
	f.afterSync = func(st *boxstation.Station, _ map[string]any) { st.ByHandle(f.calc.Handle).Child("note").Flags = "ho" }
	back := f.rollbackOf(out["batch_id"])
	if back["verdict"] != "mismatch" || !same(back["observed"], map[string]any{"restored": false}) {
		t.Errorf("back = %v", back)
	}
}

// rollbackOf is the dry run and the execution of n4_rollback of batch id.
func (f *flagsRig) rollbackOf(id any) map[string]any {
	f.T.Helper()
	return f.Srv.RunWrite("n4_rollback", Args{"batch_id": id})
}
