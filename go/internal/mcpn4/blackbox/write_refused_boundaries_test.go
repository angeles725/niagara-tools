package blackbox

// Boundaries of the refused journal state (bench retro 2026-11-15 delta 4): only the station's
// own error reply to op 1 with a read-back showing nothing applied is refused.

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// sscHook answers every callssc of sck with status and body; other frames pass.
func sscHook(sck string, status int, body func(n any, m map[string]any) []byte) func(map[string]any) (int, []byte) {
	return func(frame map[string]any) (int, []byte) {
		msgs, _ := frame["m"].([]any)
		for _, raw := range msgs {
			m, _ := raw.(map[string]any)
			if b, _ := m["b"].(map[string]any); m["k"] == "callssc" && b["sck"] == sck {
				return status, body(frame["n"], m)
			}
		}
		return 0, nil
	}
}

func lastBatchState(t *testing.T, r *Rig) any {
	t.Helper()
	lines := r.Lines("journal.jsonl")
	return batchView(t, r, lines[len(lines)-1]["batch_id"])["state"]
}

func TestALinkRefusalWithTheLinkAppliedStaysInDoubt(t *testing.T) {
	r, _, tgt, args := linkRig(t)
	r.Station.SetFaults(refuseSSC("checkLinks", "ValueError: Duplicate link already exists",
		func(st *boxstation.Station, _ map[string]any) {
			st.Component(st.ByHandle(tgt.Handle), "Link", "baja:Link", "")
		}))
	text, id := executeErr(t, r, "n4_create_link", args)
	if !strings.Contains(text, "is in-doubt") || batchView(t, r, id)["state"] != "in-doubt" {
		t.Errorf("reply %q", text)
	}
}

func TestMalformedAndHTTPFailuresOfACreateStayInDoubt(t *testing.T) {
	for name, hook := range map[string]func(map[string]any) (int, []byte){
		"empty message list": sscHook("syncTo", 200, func(any, map[string]any) []byte { return []byte(`{"v":"2.3","p":"box","n":1,"m":[]}`) }),
		"not JSON":           sscHook("syncTo", 200, func(any, map[string]any) []byte { return []byte(`<html>`) }),
		"HTTP 500":           sscHook("syncTo", 500, func(any, map[string]any) []byte { return []byte("Illegal parent") }),
	} {
		r := verifiedRig(t)
		plan := r.Srv.Dry("n4_create_component", createPump)
		r.Station.SetHook(hook)
		text := r.Srv.Err("n4_create_component", Execute(createPump, plan["confirmation_token"]))
		if st := lastBatchState(t, r); st != "in-doubt" || strings.Contains(text, "refused") {
			t.Errorf("%s: state %v reply %q", name, st, text)
		}
	}
}

func TestARefusalWhoseReadBackFailsStaysInDoubt(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetFaults(refuseSSC("syncTo", "ValueError: Illegal parent", func(st *boxstation.Station, _ map[string]any) {
		root := st.Root()
		kept := root.Children[:0]
		for _, c := range root.Children {
			if c.Name != "Folder" {
				kept = append(kept, c)
			}
		}
		root.Children = kept
	}))
	_, id := executeErr(t, r, "n4_create_component", createPump)
	if batchView(t, r, id)["state"] != "in-doubt" {
		t.Error("a failed read-back must leave the batch in-doubt")
	}
}

func TestAStationRefusalAtOp2OfACopyStaysInDoubtDecoded(t *testing.T) {
	r := verifiedRig(t)
	r.AddWithOut("Src", 1.0)
	n := 0
	r.Station.SetFaults(boxstation.Faults{Refuse: func(_ *boxstation.Station, sck string, _ map[string]any) error {
		if sck == "syncTo" {
			if n++; n == 2 {
				return errors.New("ValueError: Slot &apos;B&apos; already exists")
			}
		}
		return nil
	}})
	text, id := executeErr(t, r, "n4_copy_component", Args{"parent_ord": Folder, "name": "Src", "new_names": []any{"A", "B"}})
	if batchView(t, r, id)["state"] != "in-doubt" || !strings.Contains(text, "Slot 'B' already exists; op 2 of 2 failed") {
		t.Errorf("reply %q", text)
	}
}

func TestARefusedThenRetriedCreateRollsBackOnlyTheRetry(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetFaults(refuseSSC("syncTo", "ValueError: Illegal parent &lt;x&gt;", nil))
	_, refusedID := executeErr(t, r, "n4_create_component", createPump)
	r.Station.SetHook(nil)
	okID := r.Srv.RunWrite("n4_create_component", createPump)["batch_id"]
	if back := r.Srv.Err("n4_rollback", Args{"batch_id": refusedID}); !strings.Contains(back, "nothing to roll back") {
		t.Errorf("rollback of the refused batch = %q", back)
	}
	if rb := r.Srv.RunWrite("n4_rollback", Args{"batch_id": okID}); rb["verdict"] != "verified" || len(r.Children()) != 0 {
		t.Errorf("rollback of the retry = %v, children %v", rb["verdict"], r.Children())
	}
	retro := r.Srv.OK("n4_session_retro_draft", Args{})
	b, _ := json.Marshal(retro["candidates"])
	if md, _ := retro["markdown"].(string); !strings.Contains(md, "Journaled batches: 3 (refused 1, verified 2)") ||
		strings.Contains(string(b), "in_doubt") {
		t.Errorf("retro = %s / %s", md, b)
	}
}

func TestLinkPrecheckLeavesUndescribedAndDynamicSlotsToTheStation(t *testing.T) {
	r, src, _, args := linkRig(t)
	r.Station.SetRegistry(map[string]any{"kitControl:NumericConst": numericConst})
	r.Set(src.Handle, "dyn", StatusNumeric(3.0, "0"))
	r.Srv.Dry("n4_create_link", withArgs(args, Args{"source_slot": "dyn"})) // dynamic: only in the tree
	r2, _, _, args2 := linkRig(t)
	r2.Station.SetRegistry(map[string]any{"kitControl:NumericConst": map[string]any{}})
	r2.Srv.Dry("n4_create_link", withArgs(args2, Args{"source_slot": "nope"})) // {}: says nothing
	r3, _, _, args3 := linkRig(t)
	r3.Station.SetRegistry(map[string]any{
		"kitControl:NumericConst": map[string]any{"p": "kitControl:Base", "c": []any{map[string]any{"st": "a", "n": "set"}}},
		"kitControl:Base":         map[string]any{"c": []any{map[string]any{"st": "p", "n": "in10"}, map[string]any{"st": "p", "n": "out"}}}})
	r3.Srv.Dry("n4_create_link", args3)                                       // declared by the supertype
	r3.Srv.Dry("n4_create_link", withArgs(args3, Args{"target_slot": "set"})) // an action
	refusedDry(t, r3, "n4_create_link", withArgs(args3, Args{"target_slot": "in77"}), "target slot", "in77", "does not exist")
}

func TestAnIdenticalLinkNeedsTheSameSourceAndBothSlots(t *testing.T) {
	r, src, tgt, args := linkRig(t)
	r.Link(r.AddWithOut("Other", 2.0), "out", tgt, "in10")
	r.Link(src, "out", tgt, "in11")
	r.Srv.Dry("n4_create_link", args)
	r.Srv.Dry("n4_create_link", withArgs(args, Args{"source_slot": "in10", "target_slot": "out"}))
}
