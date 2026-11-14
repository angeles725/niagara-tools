package blackbox

// n4_create_link (port of test_tools_write TestCreateLink and TestCreateLinkRejected).

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// linkRig is the link tests' setUp: a verified session, Src (out holds 1.0) and Tgt, and the
// arguments linking Src.out to Tgt.in10.
func linkRig(t *testing.T) (r *Rig, src, tgt Component, args Args) {
	r = verifiedRig(t)
	src, tgt = r.AddWithOut("Src", 1.0), r.Add("3", "Tgt", "")
	return r, src, tgt, Args{"source_ord": Folder + "/" + src.Name, "source_slot": "out",
		"target_ord": Folder + "/" + tgt.Name, "target_slot": "in10"}
}

// replyWith makes checkLinks answer reply whatever the arguments.
func replyWith(reply any) boxstation.Faults {
	return boxstation.Faults{CheckLink: func(*boxstation.Station, map[string]any, func(map[string]any) any) any {
		return reply
	}}
}

var refusedLink = []any{map[string]any{"v": false, "r": "type mismatch", "s": nil}}

func TestHappyPathIsVerifiedAndInverseRemovesTheLink(t *testing.T) {
	r, src, tgt, args := linkRig(t)
	plan := r.Srv.Dry("n4_create_link", args)
	wantOps := []any{map[string]any{"ssc": "checkLinks", "arg": map[string]any{
		"s": src.Handle, "ss": "out", "t": tgt.Handle, "ts": "in10", "c": true}}}
	if !reflect.DeepEqual(Dig(plan, "plan", "ops"), wantOps) || hasChild(r, tgt.Handle, "Link") {
		t.Errorf("plan = %v, Tgt has a Link %v", plan["plan"], hasChild(r, tgt.Handle, "Link"))
	}
	out := r.Srv.RunWrite("n4_create_link", args)
	wantObserved := map[string]any{"link": "Link", "sourceOrd": "h:" + src.Handle, "sourceSlotName": "out",
		"targetSlotName": "in10"}
	wantInverse := []any{map[string]any{"nm": "v", "h": tgt.Handle, "n": "Link"}}
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["observed"], wantObserved) ||
		!reflect.DeepEqual(out["inverse"], wantInverse) {
		t.Errorf("out = %v", out)
	}
}

func TestBadSlotNamesAreRefused(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Srv.Err("n4_create_link", withArgs(args, Args{"target_slot": "in 10"}))
	if hasChild(r, tgt.Handle, "Link") {
		t.Error("Tgt has a Link")
	}
}

func TestCreateLinkReadBackMismatchIsReported(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Station.SetFaults(boxstation.Faults{CheckLink: func(st *boxstation.Station, arg map[string]any, orig func(map[string]any) any) any {
		res := orig(arg)
		name, _ := Dig(res.([]any)[0], "s").(string)
		st.ByHandle(tgt.Handle).Child(name).Child("sourceSlotName").Value = "other"
		return res
	}})
	if out := r.Srv.RunWrite("n4_create_link", args); out["verdict"] != "mismatch" {
		t.Errorf("out = %v", out)
	}
}

func TestStationRejectionIsFailedWithReasonAndNothingJournaled(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Station.SetFaults(replyWith(refusedLink))
	out := r.Srv.RunWrite("n4_create_link", args)
	if out["verdict"] != "failed" || Dig(out, "accepted", "r") != "type mismatch" ||
		!reflect.DeepEqual(out["inverse"], []any{}) || hasChild(r, tgt.Handle, "Link") {
		t.Errorf("out = %v", out)
	}
	lines := r.Lines("journal.jsonl")
	if len(lines) != 2 {
		t.Fatalf("journal = %v", lines)
	}
	intent, result := lines[0], lines[1]
	if intent["phase"] != "intent" || result["phase"] != "result" || result["verdict"] != "failed" ||
		!reflect.DeepEqual(result["inverse"], []any{}) {
		t.Errorf("journal = %v", lines)
	}
	if state := batchView(t, r, out["batch_id"])["state"]; state != "completed" {
		t.Errorf("state = %v", state)
	}
}
