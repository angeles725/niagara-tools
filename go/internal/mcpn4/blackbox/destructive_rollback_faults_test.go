package blackbox

// The rollback of a remove when the station misbehaves or the budget is short (port of
// test_tools_destructive.TestRollback: skipped relinks, budget, failure mid-way, read-back).

import (
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// dropAfterAdd removes child name of the group nn after every applied add.
func (d *dRig) dropAfterAdd(nn, name string) {
	d.afterAdd(func(st *boxstation.Station, _ *boxstation.Node) {
		removeChild(st.ByHandle("3").Child(nn), name)
	})
}

func TestRelinksAreSkippedWhenAnEndIsMissing(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	d.dropAfterAdd(nn, "Tgt")
	back := d.rollback(removed["batch_id"])
	if !same(back["relinks"], map[string]any{"restored": 0, "skipped": 1}) {
		t.Errorf("relinks = %v", back["relinks"])
	}
}

func TestTheComponentBudgetIsCheckedBeforeAnythingIsSent(t *testing.T) {
	d := destructiveRig(t, nil)
	d.restart(func(o *Options) { o.MaxWrites = 4 }) // remove 1 + rollback needs 1 + 2 components + 1 relink
	nn, removed := d.removeGroup()
	text := d.Srv.Err("n4_rollback", Args{"batch_id": removed["batch_id"]})
	if !strings.Contains(text, "2 component(s)") || slices.Contains(d.children(), nn) {
		t.Errorf("text %s, children %v", text, d.children())
	}
}

func TestAFailureMidWayIsInDoubtAndListsWhatWasCreated(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, removed := d.removeGroup()
	d.f.set(func(f *faults) {
		f.replace = func(sck string, arg map[string]any) (any, error, bool) {
			return nil, refusal, syncAdd(sck, arg, "Tgt") != nil
		}
	})
	plan := d.Srv.Dry("n4_rollback", Args{"batch_id": removed["batch_id"]})
	text := d.Srv.Err("n4_rollback", Execute(Args{"batch_id": removed["batch_id"]}, plan["confirmation_token"]))
	for _, needle := range []string{"in-doubt", Folder + "/" + nn, Folder + "/" + nn + "/Src"} {
		if !strings.Contains(text, needle) {
			t.Errorf("text lacks %q: %s", needle, text)
		}
	}
	if strings.Contains(text, "/Tgt") { // the failed add is not reported as created
		t.Errorf("text reports Tgt: %s", text)
	}
	var batch any
	for _, e := range d.Lines("journal.jsonl") {
		if e["rollback_of"] == removed["batch_id"] {
			batch = e["batch_id"]
			break
		}
	}
	if view := d.batch(batch); view == nil || view["state"] != "in-doubt" {
		t.Errorf("journal view = %v", view)
	}
	if d.node(nn, "Tgt") != nil {
		t.Errorf("Tgt exists")
	}
}

func TestTheNestedReadBackChecksTypeAndAnnotationNotJustPresence(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	d.Set(d.srcHandle(nn), "wsAnnotation", wsAnnotation(5, 6, 7))
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	if v := d.rollback(removed["batch_id"])["verdict"]; v != "verified" {
		t.Errorf("untampered verdict = %v", v)
	}
	tamper := func(kind string) any {
		nn2, _ := d.group("Grp2")
		d.Set(d.srcHandle(nn2), "wsAnnotation", wsAnnotation(5, 6, 7))
		gone := d.Srv.RunWrite("n4_remove_component", removeArgs(nn2))
		d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) {
			if n.Name != "Src" {
				return
			}
			if kind == "type" {
				n.Type = "kitControl:BooleanConst"
			} else {
				removeChild(n, "wsAnnotation")
			}
		})
		defer d.afterAdd(nil)
		return d.rollback(gone["batch_id"])["verdict"]
	}
	if v := tamper("type"); v != "mismatch" {
		t.Errorf("type tampered: verdict %v", v)
	}
	if v := tamper("annotation"); v != "mismatch" {
		t.Errorf("annotation tampered: verdict %v", v)
	}
}
