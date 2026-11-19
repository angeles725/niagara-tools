package blackbox

// n4_create_link dry run states what each link will command now (delta P2 (c), retro
// mcpn4-copy-keeps-status Δ2): the source slot's current value and status, outside the plan
// hash, and a named note when an {ok} source is linked into a writable's control input.

import (
	"reflect"
	"testing"
)

// commandRig is a verified rig with Sched, a NumericSchedule whose out holds 0.0 with status
// status, and Ahu, a NumericWritable.
func commandRig(t *testing.T, status string) (*Rig, Component, Component) {
	r := verifiedRig(t)
	src := r.Add("3", "Sched", "schedule:NumericSchedule")
	r.Set(src.Handle, "out", StatusNumeric(0.0, status))
	return r, src, r.Add("3", "Ahu", "control:NumericWritable")
}

func commandArgs(src, tgt, slot string) Args {
	return Args{"source_ord": Folder + "/" + src, "source_slot": "out", "target_ord": Folder + "/" + tgt,
		"target_slot": slot}
}

// sourceEntry is one link_sources entry as the dry run states it.
func sourceEntry(args Args, value any, status string, note ...string) map[string]any {
	e := map[string]any{"source_ord": args["source_ord"], "source_slot": args["source_slot"],
		"target_ord": args["target_ord"], "target_slot": args["target_slot"], "value": value, "status": status}
	if len(note) > 0 {
		e["note"] = note[0]
	}
	return e
}

const commandNote = "this link will command " + Folder + "/Ahu/in16 to 0.0 {ok} as soon as it is made"

func TestAnOkScheduleIntoAWritableIn16IsNamedInTheDryRun(t *testing.T) {
	r, _, _ := commandRig(t, "0")
	args := commandArgs("Sched", "Ahu", "in16")
	plan := r.Srv.Dry("n4_create_link", args)
	want := []any{sourceEntry(args, 0.0, "{ok}", commandNote)}
	if !reflect.DeepEqual(plan["link_sources"], want) {
		t.Errorf("link_sources = %#v, want %#v", plan["link_sources"], want)
	}
}

func TestANullSourceShowsItsValueAndNoNote(t *testing.T) {
	r, _, _ := commandRig(t, "40")
	args := commandArgs("Sched", "Ahu", "in16")
	plan := r.Srv.Dry("n4_create_link", args)
	want := []any{sourceEntry(args, 0.0, "{null}")}
	if !reflect.DeepEqual(plan["link_sources"], want) {
		t.Errorf("link_sources = %#v, want %#v", plan["link_sources"], want)
	}
}

func TestAnOkSourceIntoANonWritableHasNoNote(t *testing.T) {
	r, _, _ := commandRig(t, "0")
	r.Add("3", "Calc", "")
	args := commandArgs("Sched", "Calc", "in10")
	plan := r.Srv.Dry("n4_create_link", args)
	want := []any{sourceEntry(args, 0.0, "{ok}")}
	if !reflect.DeepEqual(plan["link_sources"], want) {
		t.Errorf("link_sources = %#v, want %#v", plan["link_sources"], want)
	}
}

func TestABatchStatesEachEntrysSource(t *testing.T) {
	r, _, _ := commandRig(t, "0")
	r.Add("3", "Calc", "")
	r.AddWithOut("Src", 1.5)
	a, b := commandArgs("Sched", "Ahu", "in16"), commandArgs("Src", "Calc", "in10")
	plan := r.Srv.Dry("n4_create_link", Args{"links": []any{map[string]any(a), map[string]any(b)}})
	want := []any{sourceEntry(a, 0.0, "{ok}", commandNote), sourceEntry(b, 1.5, "{ok}")}
	if !reflect.DeepEqual(plan["link_sources"], want) {
		t.Errorf("link_sources = %#v, want %#v", plan["link_sources"], want)
	}
}

func TestAPlainLinksHashedPlanIsUnchanged(t *testing.T) {
	r, src, tgt, args := linkRig(t)
	plan := r.Srv.Dry("n4_create_link", args)
	want := map[string]any{"ops": []any{map[string]any{"ssc": "checkLinks", "arg": map[string]any{
		"s": src.Handle, "ss": "out", "t": tgt.Handle, "ts": "in10", "c": true}}},
		"inverse": []any{map[string]any{"nm": "v", "h": tgt.Handle, "n": "<link name assigned by the station>"}},
		"notes":   []any{}}
	for k, v := range want {
		if got := Dig(plan, "plan", k); !reflect.DeepEqual(got, v) {
			t.Errorf("plan.%s = %#v, want %#v", k, got, v)
		}
	}
}

func TestASourceThatMovesAfterTheDryRunStillExecutesWithItsToken(t *testing.T) {
	r, src, _ := commandRig(t, "0")
	args := commandArgs("Sched", "Ahu", "in16")
	plan := r.Srv.Dry("n4_create_link", args)
	r.Set(src.Handle, "out", StatusNumeric(21.5, "40"))
	out := r.Srv.OK("n4_create_link", Execute(args, plan["confirmation_token"]))
	if out["verdict"] != "verified" {
		t.Errorf("out = %v", out)
	}
}
