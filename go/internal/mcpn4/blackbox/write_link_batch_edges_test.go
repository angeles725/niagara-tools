package blackbox

// n4_create_link batch form: duplicates through other ORD spellings, the write budget, an existing
// identical link, and the single form's unchanged reply.

import (
	"reflect"
	"strings"
	"testing"
)

// A source ORD that names a slot of Src (Src/out) loads Src itself, and a trailing slash names the
// same target: both entries make one link, so the second is refused as a duplicate.
func TestANestedPathAliasOfTheSameLinkIsADuplicate(t *testing.T) {
	r, _, tgts := linkBatchRig(t)
	links := []any{
		map[string]any{"source_ord": Folder + "/Src", "source_slot": "out", "target_ord": Folder + "/T1", "target_slot": "in10"},
		map[string]any{"source_ord": Folder + "/Src/out", "source_slot": "out", "target_ord": Folder + "/T1/", "target_slot": "in10"},
	}
	text := r.Srv.Err("n4_create_link", Args{"links": links})
	if !strings.Contains(text, "twice") || !strings.Contains(text, "links[0]") || !strings.Contains(text, "links[1]") {
		t.Errorf("refusal = %s", text)
	}
	if hasChild(r, tgts[0].Handle, "Link") {
		t.Error("a refused batch linked T1")
	}
}

// An entry identical to a link the target already has is refused, as the single form refuses it.
func TestAnEntryIdenticalToAnExistingLinkRefusesTheBatch(t *testing.T) {
	r, src, tgts := linkBatchRig(t)
	r.Link(src, "out", tgts[1], "in10")
	text := r.Srv.Err("n4_create_link", Args{"links": threeLinks()})
	if !strings.Contains(text, "links[1]") || !strings.Contains(text, "identical link already exists") {
		t.Errorf("refusal = %s", text)
	}
}

// A batch of N links spends N writes of --max-writes, and a plan the budget cannot hold is refused
// before its token.
func TestALinkBatchSpendsOneWritePerLink(t *testing.T) {
	r, _, _ := linkBatchRig(t)
	out := r.Srv.RunWrite("n4_create_link", Args{"links": threeLinks()})
	if got := Dig(out, "write_budget", "executed"); got != 3.0 {
		t.Errorf("write_budget executed = %v", got)
	}
	small := writeRig(t, func(o *Options) { o.MaxWrites = 2 })
	small.ConnectVerified()
	small.AddWithOut("Src", 1.0)
	for _, n := range []string{"T1", "T2", "T3"} {
		small.Add("3", n, "")
	}
	if text := small.Srv.Err("n4_create_link", Args{"links": threeLinks()}); !strings.Contains(text, "needs 3 write(s) but only 2 remain") {
		t.Errorf("refusal = %s", text)
	}
}

// The single form keeps its plan and reply; a missing argument is a tool error naming it.
func TestTheSingleLinkFormReplyIsUnchanged(t *testing.T) {
	r, src, tgt, args := linkRig(t)
	plan := r.Srv.Dry("n4_create_link", args)
	if got := keys(plan["plan"].(map[string]any)); !reflect.DeepEqual(got, []string{"inverse", "notes", "ops", "station", "tool"}) {
		t.Errorf("plan keys = %v", got)
	}
	if notes := Dig(plan, "plan", "notes"); !reflect.DeepEqual(notes, []any{}) {
		t.Errorf("notes = %v", notes)
	}
	out := r.Srv.OK("n4_create_link", Execute(args, plan["confirmation_token"]))
	wantKeys := []string{"accepted", "batch_id", "dry_run", "inverse", "observed", "requested", "verdict", "write_budget"}
	if got := keys(out); !reflect.DeepEqual(got, wantKeys) {
		t.Errorf("keys = %v", got)
	}
	wantObserved := map[string]any{"link": "Link", "sourceOrd": "h:" + src.Handle, "sourceSlotName": "out", "targetSlotName": "in10"}
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["observed"], wantObserved) ||
		Dig(out, "write_budget", "executed") != 1.0 {
		t.Errorf("out = %v", out)
	}
	if !hasChild(r, tgt.Handle, "Link") {
		t.Error("Tgt has no Link")
	}
	missing := withArgs(args, nil)
	delete(missing, "source_ord")
	if text := r.Srv.Err("n4_create_link", missing); text != "missing required argument: source_ord" {
		t.Errorf("missing source_ord: %q", text)
	}
}
