package blackbox

// What n4_create_link makes of each checkLinks reply (port of test_tools_write TestLinkReplies).

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// journalView is the batch's journal view; every returned batch_id has one.
func journalView(t *testing.T, r *Rig, out map[string]any) map[string]any {
	t.Helper()
	view := batchView(t, r, out["batch_id"])
	if view == nil {
		t.Fatalf("batch %v has no journal entry", out["batch_id"])
	}
	return view
}

func TestAnExplicitRefusalIsAFailedResultWithAnEmptyInverse(t *testing.T) {
	r, _, _, args := linkRig(t)
	r.Station.SetFaults(replyWith(refusedLink))
	out := r.Srv.RunWrite("n4_create_link", args)
	view := journalView(t, r, out)
	if out["verdict"] != "failed" || view["verdict"] != "failed" || !reflect.DeepEqual(view["inverse"], []any{}) ||
		view["state"] != "completed" {
		t.Errorf("out = %v, view = %v", out, view)
	}
}

func TestVTrueWithoutALinkNameIsAmbiguousAndFindsTheLinkByItsSlots(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Station.SetFaults(boxstation.Faults{CheckLink: func(_ *boxstation.Station, arg map[string]any, orig func(map[string]any) any) any {
		reply := with(orig(arg).([]any)[0].(map[string]any), Args{"s": nil})
		return []any{reply}
	}})
	out := r.Srv.RunWrite("n4_create_link", args)
	view := journalView(t, r, out)
	wantInverse := []any{map[string]any{"nm": "v", "h": tgt.Handle, "n": "Link"}}
	if out["verdict"] != "unverified" || view["verdict"] != "unverified" || view["state"] != "in-doubt" ||
		out["in_doubt"] != true || Dig(out, "observed", "link") != "Link" ||
		!reflect.DeepEqual(out["inverse"], wantInverse) || !reflect.DeepEqual(view["inverse"], out["inverse"]) {
		t.Errorf("out = %v, view = %v", out, view)
	}
}

func TestAnAmbiguousReplyWithNoLinkFoundRecordsAnEmptyInverse(t *testing.T) {
	r, _, _, args := linkRig(t)
	r.Station.SetFaults(replyWith([]any{map[string]any{"v": true, "r": nil, "s": ""}}))
	out := r.Srv.RunWrite("n4_create_link", args)
	view := journalView(t, r, out)
	observed, has := out["observed"]
	if out["verdict"] != "unverified" || !has || observed != nil || !reflect.DeepEqual(out["inverse"], []any{}) ||
		view["state"] != "in-doubt" {
		t.Errorf("out = %v, view = %v", out, view)
	}
}

func TestAMalformedReplyIsInDoubtNotACrash(t *testing.T) {
	r, _, _, args := linkRig(t)
	for _, junk := range []any{"junk", []any{}, []any{nil}, []any{[]any{}}} {
		r.Station.SetFaults(replyWith(junk))
		out := r.Srv.RunWrite("n4_create_link", args)
		view := journalView(t, r, out)
		if _, has := out["readback_error"]; out["verdict"] != "unverified" || view["state"] != "in-doubt" || !has {
			t.Errorf("%v: out = %v, view = %v", junk, out, view)
		}
	}
}
