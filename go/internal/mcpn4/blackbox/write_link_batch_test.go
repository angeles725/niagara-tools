package blackbox

// n4_create_link batch form (field retro delta U11): several links under one dry run and
// one confirmation token, each read back.

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// linkBatchRig is a verified rig with Src (out 1.0) and three targets T1..T3.
func linkBatchRig(t *testing.T) (*Rig, Component, []Component) {
	r := verifiedRig(t)
	src := r.AddWithOut("Src", 1.0)
	tgts := []Component{r.Add("3", "T1", ""), r.Add("3", "T2", ""), r.Add("3", "T3", "")}
	return r, src, tgts
}

// threeLinks link Src.out to in10 of T1, T2 and T3.
func threeLinks() []any {
	out := []any{}
	for _, n := range []string{"T1", "T2", "T3"} {
		out = append(out, map[string]any{"source_ord": Folder + "/Src", "source_slot": "out",
			"target_ord": Folder + "/" + n, "target_slot": "in10"})
	}
	return out
}

func TestABatchOfThreeLinksIsOneDryRunOneTokenAndVerified(t *testing.T) {
	r, src, tgts := linkBatchRig(t)
	args := Args{"links": threeLinks()}
	plan := r.Srv.Dry("n4_create_link", args)
	ops, _ := Dig(plan, "plan", "ops").([]any)
	if len(ops) != 3 || Dig(ops[2], "arg", "t") != tgts[2].Handle || Dig(ops[0], "arg", "s") != src.Handle {
		t.Errorf("ops = %v", ops)
	}
	if hasChild(r, tgts[0].Handle, "Link") {
		t.Error("the dry run linked T1")
	}
	out := r.Srv.OK("n4_create_link", Execute(args, plan["confirmation_token"]))
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	if got := field(out["observed"], "verdict"); !reflect.DeepEqual(got, []any{"verified", "verified", "verified"}) {
		t.Errorf("entry verdicts = %v", got)
	}
	wantInverse := []any{}
	for _, c := range tgts {
		wantInverse = append(wantInverse, map[string]any{"nm": "v", "h": c.Handle, "n": "Link"})
		if !hasChild(r, c.Handle, "Link") {
			t.Errorf("%s has no Link", c.Name)
		}
	}
	if !reflect.DeepEqual(out["inverse"], wantInverse) {
		t.Errorf("inverse = %v", out["inverse"])
	}
	if got := phases(r); !reflect.DeepEqual(got, []any{"intent", "result"}) {
		t.Errorf("journal phases = %v", got)
	}
}

// One link the station accepted but that is gone at the read-back makes the batch partial.
func TestAMissingLinkAtTheReadBackMakesTheBatchPartial(t *testing.T) {
	r, _, tgts := linkBatchRig(t)
	r.Station.SetFaults(boxstation.Faults{CheckLink: func(st *boxstation.Station, arg map[string]any, orig func(map[string]any) any) any {
		res := orig(arg)
		if arg["t"] == tgts[1].Handle {
			n := st.ByHandle(tgts[1].Handle)
			kept := n.Children[:0]
			for _, c := range n.Children {
				if c.Name != "Link" {
					kept = append(kept, c)
				}
			}
			n.Children = kept
		}
		return res
	}})
	out := r.Srv.RunWrite("n4_create_link", Args{"links": threeLinks()})
	verdicts := field(out["observed"], "verdict")
	if out["verdict"] != "partial" || !reflect.DeepEqual(verdicts, []any{"verified", "mismatch", "verified"}) {
		t.Errorf("verdict %v, entries %v", out["verdict"], verdicts)
	}
	if observed := objects(out["observed"])[1]["observed"]; observed != nil {
		t.Errorf("T2 observed = %v", observed)
	}
}

func TestTheSameLinkTwiceInABatchIsRefused(t *testing.T) {
	r, _, tgts := linkBatchRig(t)
	links := threeLinks()
	links[2] = links[0]
	text := r.Srv.Err("n4_create_link", Args{"links": links})
	if !strings.Contains(text, "twice") || !strings.Contains(text, "links[0]") || !strings.Contains(text, "links[2]") {
		t.Errorf("refusal = %s", text)
	}
	if hasChild(r, tgts[0].Handle, "Link") {
		t.Error("a refused batch linked T1")
	}
}

func TestMixingTheSingleFormAndLinksIsRefused(t *testing.T) {
	r, _, _ := linkBatchRig(t)
	for _, extra := range []string{"source_ord", "source_slot", "target_ord", "target_slot"} {
		args := Args{"links": threeLinks(), extra: Folder + "/T1"}
		if text := r.Srv.Err("n4_create_link", args); !strings.Contains(text, "links") || !strings.Contains(text, "not both") {
			t.Errorf("%s: refusal = %s", extra, text)
		}
	}
}

func TestZeroOrMoreThanFiftyLinksAreRefused(t *testing.T) {
	r, _, _ := linkBatchRig(t)
	many := []any{}
	for i := 0; i < 51; i++ {
		many = append(many, map[string]any{"source_ord": Folder + "/Src", "source_slot": "out",
			"target_ord": Folder + "/T1", "target_slot": fmt.Sprintf("in%d", i)})
	}
	for _, links := range [][]any{{}, many} {
		if text := r.Srv.Err("n4_create_link", Args{"links": links}); !strings.Contains(text, "1 to 50") {
			t.Errorf("%d entries: refusal = %s", len(links), text)
		}
	}
}

// The schema lists an entry's four arguments as required: a missing one is a -32602 error naming it.
func TestAnEntryMissingAnArgumentIsRefusedNamingIt(t *testing.T) {
	r, _, _ := linkBatchRig(t)
	links := threeLinks()
	delete(links[1].(map[string]any), "target_slot")
	res := r.Srv.Request("tools/call", Args{"name": "n4_create_link", "arguments": Args{"links": links}})
	if msg, _ := Dig(res, "error", "message").(string); ErrorCode(res) != -32602 || !strings.Contains(msg, "links[1].target_slot") {
		t.Errorf("reply = %v", res)
	}
}

func TestOneEntryOutsideTheWriteScopeRefusesTheWholeLinkBatch(t *testing.T) {
	r := writeRig(t, func(o *Options) { o.WriteScopes = []string{Folder + "/T1"} })
	r.ConnectVerified()
	r.AddWithOut("Src", 1.0)
	t1 := r.Add("3", "T1", "")
	r.Add("3", "T2", "")
	r.Add("3", "T3", "")
	if text := r.Srv.Err("n4_create_link", Args{"links": threeLinks()}); !strings.Contains(text, "outside the write scope") ||
		!strings.Contains(text, "links[1]") {
		t.Errorf("refusal = %s", text)
	}
	if hasChild(r, t1.Handle, "Link") {
		t.Error("a refused batch linked T1")
	}
}

func TestRollbackOfALinkBatchRemovesEveryLink(t *testing.T) {
	r, _, tgts := linkBatchRig(t)
	out := r.Srv.RunWrite("n4_create_link", Args{"links": threeLinks()})
	if out["verdict"] != "verified" {
		t.Fatalf("out = %v", out)
	}
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]})
	if back["verdict"] != "verified" {
		t.Errorf("rollback = %v", back)
	}
	for _, c := range tgts {
		if hasChild(r, c.Handle, "Link") {
			t.Errorf("%s still has its Link", c.Name)
		}
	}
}
