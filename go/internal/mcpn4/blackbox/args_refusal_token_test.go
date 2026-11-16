package blackbox

// A write refused by the shared argument validator (retro row M11) never consumes its
// confirmation token nor reaches the station; the same token then executes.

import "testing"

func TestRefusedArgumentDoesNotConsumeTheConfirmationToken(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	args := Args{"parent_ord": Folder, "name": "Probe", "type": "baja:Folder"}
	plan := h.Srv.Dry("n4_create_component", withArgs(args, Args{"dry_run": true}))
	tok := plan["confirmation_token"]
	if tok == nil || tok == "" {
		t.Fatalf("no token: %v", plan)
	}
	before := h.Station.Requests()
	for _, bad := range []Args{{"zz_unknown": true}, {"confirm": true}} {
		reply := h.Srv.Request("tools/call", Args{"name": "n4_create_component",
			"arguments": withArgs(Execute(args, tok), bad)})
		if ErrorCode(reply) != -32602 {
			t.Fatalf("%v: want -32602, got %v", bad, reply)
		}
	}
	if got := h.Station.Requests(); got != before {
		t.Fatalf("a refused executing call reached the station: %d requests, before %d", got, before)
	}
	out := h.Srv.OK("n4_create_component", Execute(args, tok))
	if out["verdict"] != "verified" {
		t.Errorf("the token was spent by a refused call: %v", out)
	}
}

func TestRefusedEnumOnAWriteDoesNotConsumeTheConfirmationToken(t *testing.T) {
	h := newHistoryRig(t, noHistoryIDs)
	h.Add("3", "Temp", "control:NumericWritable")
	args := historyArgs(nil, "Temp")
	plan := h.Srv.Dry("n4_add_history", withArgs(args, Args{"dry_run": true}))
	tok := plan["confirmation_token"]
	before := h.Station.Requests()
	reply := h.Srv.Request("tools/call", Args{"name": "n4_add_history",
		"arguments": withArgs(Execute(args, tok), Args{"full_policy": "Roll"})})
	if ErrorCode(reply) != -32602 || h.Station.Requests() != before {
		t.Fatalf("want -32602 and no request, got %v (%d requests)", reply, h.Station.Requests()-before)
	}
	if out := h.Srv.OK("n4_add_history", Execute(args, tok)); out["verdict"] != "verified" {
		t.Errorf("the token was spent by a refused call: %v", out)
	}
}
