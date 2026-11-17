package blackbox

// n4_set_slot batch form (overamps retro row M6, backlog OD040): several (ord, slot, value)
// entries under one dry run and one confirmation token.

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// batchRig is a verified rig with two components: A (out 1.0, d 2.0) and B (out 3.0).
func batchRig(t *testing.T) (*Rig, Component, Component) {
	r := verifiedRig(t)
	a, b := r.AddWithOut("A", 1.0), r.AddWithOut("B", 3.0)
	r.Set(a.Handle, "d", map[string]any{"nm": "p", "t": "baja:Double", "v": "2.0"})
	return r, a, b
}

// threeWrites are three entries on two components.
func threeWrites() []any {
	return []any{
		map[string]any{"ord": Folder + "/A", "slot": "out", "value": 5.5, "value_type": "baja:StatusNumeric"},
		map[string]any{"ord": Folder + "/A", "slot": "d", "value": 4.5, "value_type": "baja:Double"},
		map[string]any{"ord": Folder + "/B", "slot": "out", "value": 7.0, "value_type": "baja:StatusNumeric", "status": "0"},
	}
}

func TestABatchOfThreeWritesIsOneDryRunOneTokenAndVerified(t *testing.T) {
	r, a, b := batchRig(t)
	args := Args{"writes": threeWrites()}
	plan := r.Srv.Dry("n4_set_slot", args)
	ops, _ := Dig(plan, "plan", "ops").([]any)
	want := append(append(setOp(a.Handle, "out", StatusNumeric(5.5, "0")),
		setOp(a.Handle, "d", map[string]any{"nm": "p", "t": "baja:Double", "v": "4.5"})...),
		setOp(b.Handle, "out", StatusNumeric(7.0, "0"))...)
	if !reflect.DeepEqual(ops, want) {
		t.Errorf("ops = %v", ops)
	}
	out := r.Srv.OK("n4_set_slot", Execute(args, plan["confirmation_token"]))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	observed, _ := out["observed"].([]any)
	wantObserved := []any{
		map[string]any{"ord": Folder + "/A", "slot": "out", "requested": map[string]any{"value": 5.5, "status": "0"},
			"observed": map[string]any{"value": 5.5, "status": "0"}, "verdict": "verified"},
		map[string]any{"ord": Folder + "/A", "slot": "d", "requested": 4.5, "observed": 4.5, "verdict": "verified"},
		map[string]any{"ord": Folder + "/B", "slot": "out", "requested": map[string]any{"value": 7.0, "status": "0"},
			"observed": map[string]any{"value": 7.0, "status": "0"}, "verdict": "verified"},
	}
	if !reflect.DeepEqual(observed, wantObserved) {
		t.Errorf("observed = %v", observed)
	}
	if got := Dig(out, "write_budget", "executed"); got != 1.0 { // one batch, one write, as n4_set_tags
		t.Errorf("write_budget executed = %v", got)
	}
	if got := phases(r); !reflect.DeepEqual(got, []any{"intent", "result"}) { // one batch
		t.Errorf("journal phases = %v", got)
	}
}

func TestABatchEntryMismatchMakesTheWholeBatchAMismatch(t *testing.T) {
	r, a, _ := batchRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		st.ByHandle(a.Handle).Child("d").Value = "9.0"
	}})
	out := r.Srv.RunWrite("n4_set_slot", Args{"writes": threeWrites()})
	verdicts := field(out["observed"], "verdict")
	if out["verdict"] != "mismatch" || !reflect.DeepEqual(verdicts, []any{"verified", "mismatch", "verified"}) {
		t.Errorf("verdict %v, entries %v", out["verdict"], verdicts)
	}
}

// Each entry's incoming links reach the batch plan, with the entry's ord as target_ord, and its
// note names the entry.
func TestABatchPlanListsTheIncomingLinksOfEachEntry(t *testing.T) {
	r, a, b := batchRig(t)
	r.Link(a, "out", b, "out")
	plan := r.Srv.Dry("n4_set_slot", Args{"writes": threeWrites()})
	links, _ := Dig(plan, "plan", "incoming_links").([]any)
	if len(links) != 1 || Dig(links[0], "target_ord") != Folder+"/B" || Dig(links[0], "target_slot") != "out" {
		t.Errorf("incoming_links = %v", links)
	}
	if notes := pyjsonText(Dig(plan, "plan", "notes")); !strings.Contains(notes, "writes[2] ("+Folder+"/B out): 1 link(s)") {
		t.Errorf("notes = %s", notes)
	}
}

func TestMixingTheSingleFormAndWritesIsRefused(t *testing.T) {
	r, _, _ := batchRig(t)
	for extra, v := range map[string]any{"ord": Folder + "/A", "slot": "out", "value": 1.0,
		"value_type": "baja:Double", "status": "0", "simple_type": "baja:String"} {
		args := Args{"writes": threeWrites(), extra: v}
		if text := r.Srv.Err("n4_set_slot", args); !strings.Contains(text, "writes") || !strings.Contains(text, "not both") {
			t.Errorf("%s: refusal = %s", extra, text)
		}
	}
}

func TestTheSameOrdAndSlotTwiceIsRefused(t *testing.T) {
	r, a, _ := batchRig(t)
	writes := threeWrites()
	writes[2] = map[string]any{"ord": Folder + "/A", "slot": "out", "value": 8.0, "value_type": "baja:StatusNumeric"}
	if text := r.Srv.Err("n4_set_slot", Args{"writes": writes}); !strings.Contains(text, "twice") {
		t.Errorf("refusal = %s", text)
	}
	if got := nodeValue(r, a.Handle, "out", "value"); got != "1.0" {
		t.Errorf("A.out = %v", got)
	}
}

func TestOneEntryOutsideTheWriteScopeRefusesTheWholeBatch(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{Folder + "/A"} })
	r.ConnectVerified()
	a, b := r.AddWithOut("A", 1.0), r.AddWithOut("B", 3.0)
	r.Set(a.Handle, "d", map[string]any{"nm": "p", "t": "baja:Double", "v": "2.0"})
	text := r.Srv.Err("n4_set_slot", Execute(Args{"writes": threeWrites()}, "1.a.b"))
	if !strings.Contains(text, "outside the write scope") {
		t.Errorf("refusal = %s", text)
	}
	if text := r.Srv.Err("n4_set_slot", Args{"writes": threeWrites()}); !strings.Contains(text, "outside the write scope") {
		t.Errorf("dry run refusal = %s", text)
	}
	if nodeValue(r, a.Handle, "out", "value") != "1.0" || nodeValue(r, b.Handle, "out", "value") != "3.0" {
		t.Error("a refused batch changed the station")
	}
}

func TestAChangedEntryValueAfterTheDryRunRefusesTheToken(t *testing.T) {
	r, a, b := batchRig(t)
	plan := r.Srv.Dry("n4_set_slot", Args{"writes": threeWrites()})
	for i := range threeWrites() {
		changed := threeWrites()
		changed[i].(map[string]any)["value"] = 6.25
		text := r.Srv.Err("n4_set_slot", Execute(Args{"writes": changed}, plan["confirmation_token"]))
		if !strings.Contains(text, "confirmation_token") {
			t.Errorf("entry %d changed: refusal = %s", i, text)
		}
	}
	if nodeValue(r, a.Handle, "out", "value") != "1.0" || nodeValue(r, b.Handle, "out", "value") != "3.0" ||
		nodeValue(r, a.Handle, "d") != "2.0" {
		t.Error("a refused token changed the station")
	}
}

func TestRollbackOfABatchRestoresEveryEntry(t *testing.T) {
	r, a, b := batchRig(t)
	set := r.Srv.RunWrite("n4_set_slot", Args{"writes": threeWrites()})
	if set["verdict"] != "verified" || nodeValue(r, a.Handle, "d") != "4.5" {
		t.Fatalf("set = %v", set)
	}
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": set["batch_id"]})
	if back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	for _, c := range []struct {
		h    string
		path []string
		want string
	}{{a.Handle, []string{"out", "value"}, "1.0"}, {a.Handle, []string{"d"}, "2.0"}, {b.Handle, []string{"out", "value"}, "3.0"}} {
		if got := nodeValue(r, c.h, c.path...); got != c.want {
			t.Errorf("%s %v = %v, want %s", c.h, c.path, got, c.want)
		}
	}
}

func TestMoreThanFiftyWritesAreRefused(t *testing.T) {
	r, _, _ := batchRig(t)
	writes := []any{}
	for i := 0; i < 51; i++ {
		writes = append(writes, map[string]any{"ord": Folder + "/A", "slot": fmt.Sprintf("s%d", i), "value": 1.0,
			"value_type": "baja:Double"})
	}
	if text := r.Srv.Err("n4_set_slot", Args{"writes": writes}); !strings.Contains(text, "1 to 50") {
		t.Errorf("refusal = %s", text)
	}
	if text := r.Srv.Err("n4_set_slot", Args{"writes": []any{}}); !strings.Contains(text, "1 to 50") {
		t.Errorf("empty: refusal = %s", text)
	}
}
