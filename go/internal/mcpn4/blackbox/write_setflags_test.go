package blackbox

// n4_set_flags: the flags of one slot of a component, with the dry run, token, journal and
// read-back of every write.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// flagsAccepted is the station's reply to a set-flags op (syncToMaster: one entry per op, its id).
var flagsAccepted = []any{map[string]any{"id": "f"}}

const flagsUnprovenNote = "setting slot flags is not yet proven on a live station: try it on a scratch slot first"

func TestSetFlagsDryRunIsTheDefaultShowsBothSetsAndTouchesNothing(t *testing.T) {
	f := newFlagsRig(t, "h")
	reply := f.Srv.OK("n4_set_flags", f.args("oh")) // no dry_run argument
	token, _ := reply["confirmation_token"].(string)
	if reply["dry_run"] != true || token == "" {
		t.Errorf("reply = %v", reply)
	}
	plan := planOf(reply)
	if !same(plan["ops"], []any{f.op("ho")}) || !same(plan["inverse"], []any{f.op("h")}) || plan["tool"] != "n4_set_flags" {
		t.Errorf("plan = %v", plan)
	}
	wantNotes := []any{"flags of slot 'note' on " + f.ord + `: current "h" (h hidden) -> requested "ho" (h hidden, o operator)`,
		flagsUnprovenNote}
	if !reflect.DeepEqual(plan["notes"], wantNotes) {
		t.Errorf("notes\n got  %v\n want %v", plan["notes"], wantNotes)
	}
	f.untouched(t, "h")
	audit := f.Lines("audit.jsonl")
	if last := audit[len(audit)-1]; last["tool"] != "n4_set_flags" || last["outcome"] != "planned" || last["dry_run"] != true {
		t.Errorf("audit = %v", last)
	}
}

func TestSetFlagsExecutesWithTheTokenAndVerifiesByReadBack(t *testing.T) {
	f := newFlagsRig(t, "h")
	out := f.Srv.RunWrite("n4_set_flags", f.args("oh"))
	if out["dry_run"] != false || out["verdict"] != "verified" || out["requested"] != "ho" || out["observed"] != "ho" ||
		!same(out["accepted"], flagsAccepted) || !same(out["inverse"], []any{f.op("h")}) {
		t.Errorf("out = %v", out)
	}
	if got := f.sent(); !same(got, []any{f.op("ho")}) { // the one op the station received
		t.Errorf("sent = %v", got)
	}
	if got := f.flags(); got != "ho" {
		t.Errorf("flags = %q", got)
	}
	lines := f.Lines("journal.jsonl")
	if len(lines) != 2 {
		t.Fatalf("journal = %v", lines)
	}
	intent, result := lines[0], lines[1]
	if intent["phase"] != "intent" || intent["tool"] != "n4_set_flags" || intent["batch_id"] != out["batch_id"] ||
		!same(intent["ops"], []any{f.op("ho")}) || !same(intent["inverse_plan"], []any{f.op("h")}) ||
		!same(intent["targets"], map[string]any{f.ord: f.calc.Handle}) || intent["station_name"] != StationName {
		t.Errorf("intent = %v", intent)
	}
	if result["phase"] != "result" || result["batch_id"] != out["batch_id"] || result["verdict"] != "verified" ||
		!same(result["inverse"], []any{f.op("h")}) || !same(result["accepted"], flagsAccepted) {
		t.Errorf("result = %v", result)
	}
	audit := f.Lines("audit.jsonl")
	if last := audit[len(audit)-1]; last["outcome"] != "executed" || last["reason"] != "verdict=verified" ||
		last["batch_id"] != out["batch_id"] {
		t.Errorf("audit = %v", last)
	}
}

func TestSetFlagsClearsEveryFlagWithAnEmptyString(t *testing.T) {
	f := newFlagsRig(t, "ho")
	out := f.Srv.RunWrite("n4_set_flags", f.args(""))
	if out["verdict"] != "verified" || out["requested"] != "" || out["observed"] != "" || f.flags() != "" ||
		!same(f.sent(), []any{f.op("")}) || !same(out["inverse"], []any{f.op("ho")}) {
		t.Errorf("out = %v, flags %q, sent %v", out, f.flags(), f.sent())
	}
}

func TestSetFlagsExecutionNeedsTheTokenOfItsOwnPlan(t *testing.T) {
	f := newFlagsRig(t, "h")
	if text := f.Srv.Err("n4_set_flags", withArgs(f.args("ho"), Args{"dry_run": false})); !strings.Contains(text, "confirmation") {
		t.Errorf("no token: %s", text)
	}
	token := f.Srv.Dry("n4_set_flags", f.args("ho"))["confirmation_token"]
	if text := f.Srv.Err("n4_set_flags", Execute(f.args("hs"), token)); !strings.Contains(text, "does not match") {
		t.Errorf("another flag set: %s", text)
	}
	f.setFlags("r") // someone else changes the flags: the inverse of the plan is no longer true
	token = f.Srv.Dry("n4_set_flags", f.args("ho"))["confirmation_token"]
	f.setFlags("h")
	if text := f.Srv.Err("n4_set_flags", Execute(f.args("ho"), token)); !strings.Contains(text, "this plan") {
		t.Errorf("flags changed since the dry run: %s", text)
	}
	f.untouched(t, "h")
}

// The op's reply is not the verdict: a station that answers the op and holds other flags is a
// mismatch with both sets, and the batch is journaled as such.
func TestSetFlagsReadBackMismatchIsReportedWithRequestedAndObserved(t *testing.T) {
	f := newFlagsRig(t, "h")
	f.afterSync = func(st *boxstation.Station, _ map[string]any) { st.ByHandle(f.calc.Handle).Child("note").Flags = "r" }
	out := f.Srv.RunWrite("n4_set_flags", f.args("ho"))
	if out["verdict"] != "mismatch" || out["requested"] != "ho" || out["observed"] != "r" ||
		!same(out["accepted"], flagsAccepted) {
		t.Errorf("out = %v", out)
	}
	lines := f.Lines("journal.jsonl")
	if len(lines) != 2 || lines[1]["verdict"] != "mismatch" || !same(lines[1]["inverse"], []any{f.op("h")}) {
		t.Errorf("journal = %v", lines)
	}
}

func TestSetFlagsRefusalsSendNothingAndJournalNothing(t *testing.T) {
	f := newFlagsRig(t, "ho")
	for _, tc := range []struct {
		name    string
		args    Args
		needles []string
	}{
		{"an unknown letter", f.args("hq"), []string{"unknown flag letter 'q'", "valid letters: r readonly, t transient", "4 user-defined-4"}},
		{"a repeated letter", f.args("hoh"), []string{"flag letter 'h' is repeated in flags 'hoh'"}},
		{"a slot the component does not have", withArgs(f.args("h"), Args{"slot": "nope"}),
			[]string{"slot 'nope' does not exist on " + f.ord, "n4_read_slots"}},
		{"a nested slot path", withArgs(f.args("h"), Args{"slot": "out/value"}), []string{"slot must be a plain slot name"}},
		{"an ord outside the write scope", Args{"ord": "station:|slot:/", "slot": "stationName", "flags": "h"},
			[]string{"outside the write scope"}},
		{"the set the slot already has", f.args("oh"), []string{"already has flags \"ho\" (h hidden, o operator): nothing to do"}},
	} {
		for _, dry := range []bool{true, false} { // refused as a dry run and as an execution
			text := f.Srv.Err("n4_set_flags", withArgs(tc.args, Args{"dry_run": dry, "confirmation_token": "1.a.b"}))
			for _, needle := range tc.needles {
				if !strings.Contains(text, needle) {
					t.Errorf("%s (dry_run %v): refusal lacks %q: %s", tc.name, dry, needle, text)
				}
			}
		}
	}
	f.untouched(t, "ho")
	audit := f.Lines("audit.jsonl")
	if last := audit[len(audit)-1]; last["tool"] != "n4_set_flags" || last["outcome"] != "refused" {
		t.Errorf("audit = %v", last)
	}
}

// A slot the station lists without flags may be a frozen slot at its declared flags: when the
// type contract cannot be read there is no faithful inverse, so nothing is planned.
func TestSetFlagsRefusesWhenTheCurrentFlagsCannotBeDetermined(t *testing.T) {
	f := newFlagsRig(t, "")
	f.noTypes()
	text := f.Srv.Err("n4_set_flags", f.args("h"))
	for _, needle := range []string{"cannot determine the current flags of slot 'note' on " + f.ord,
		"the station did not describe type 'kitControl:NumericConst'", "no faithful inverse"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if got := f.flags(); got != "" || len(f.Lines("journal.jsonl")) != 0 {
		t.Errorf("flags %q, journal %v", got, f.Lines("journal.jsonl"))
	}
}

func TestSetFlagsIsListedAsAWriteToolOnlyWhenWritesAreAllowed(t *testing.T) {
	tool := listedTools(writeRig(t, nil).Srv)["n4_set_flags"]
	if tool == nil {
		t.Fatal("n4_set_flags is not listed by a writes-allowed server")
	}
	ann := annotations(t, tool, "n4_set_flags") // also checks dry_run and confirmation_token
	if ann["readOnlyHint"] != false || ann["destructiveHint"] != false || ann["openWorldHint"] != false || ann["idempotentHint"] != true {
		t.Errorf("annotations = %v", ann)
	}
	if def := Dig(tool, "inputSchema", "properties", "dry_run", "default"); def != true {
		t.Errorf("dry_run default = %v", def)
	}
	if req := Dig(tool, "inputSchema", "required"); !reflect.DeepEqual(req, []any{"ord", "slot", "flags"}) {
		t.Errorf("required = %v", req)
	}
	r := NewRig(t, nil)
	if _, ok := listedTools(r.Srv)["n4_set_flags"]; ok {
		t.Error("n4_set_flags is listed by a read-only server")
	}
	if code := r.Srv.CallError("n4_set_flags", Args{}); code != -32601 {
		t.Errorf("read-only call: code %d, want -32601", code)
	}
}
