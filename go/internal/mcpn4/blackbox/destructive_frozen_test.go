package blackbox

// Frozen children: a writable's proxyExt is created with its parent, never re-added, and what
// it held is compared after the rollback (port of test_tools_destructive.TestRollback, issue
// #179 R4-002 / B1200-G1).

import (
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// writableGroup is a group also holding Sp (NumericWritable) and En (BooleanWritable), each with
// its frozen proxyExt.
func (d *dRig) writableGroup() string {
	d.T.Helper()
	nn, gh := d.group("Grp")
	d.Add(gh, "Sp", "control:NumericWritable")
	d.Add(gh, "En", "control:BooleanWritable")
	return nn
}

// configureProxyExt gives Sp's frozen proxyExt a configured (non-default) slot, as a real point has.
func (d *dRig) configureProxyExt(nn string) {
	d.T.Helper()
	d.Set(d.node(nn, "Sp", "proxyExt").Handle, "tuningPolicyName", map[string]any{"nm": "p", "t": "baja:String", "v": "Fast"})
}

// pathTail is the last k segments of a "/" path.
func pathTail(path any, k int) []string {
	s, _ := path.(string)
	parts := strings.Split(s, "/")
	if len(parts) < k {
		return parts
	}
	return parts[len(parts)-k:]
}

func TestAGroupWithWritablesRollsBackWithoutReAddingFrozenChildren(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if got := names(adds(sent())); !same(got, []any{nn, "Src", "Tgt", "Sp", "En"}) {
		t.Errorf("adds = %v", got)
	}
	for _, name := range []string{"Sp", "En"} {
		var types []string
		comp := d.node(nn, name)
		if comp == nil {
			t.Fatalf("%s was not re-created", name)
		}
		d.Station.Do(func(*boxstation.Station) {
			for _, c := range comp.Children {
				if c.Name == "proxyExt" {
					types = append(types, c.Type)
				}
			}
		})
		if !slices.Equal(types, []string{"control:NullProxyExt"}) {
			t.Errorf("%s proxyExt types = %v", name, types)
		}
	}
	frozen := objects(back["frozen_children_not_restored"])
	var pairs, parents []string
	for _, f := range frozen {
		pairs = append(pairs, f["n"].(string)+"|"+f["type"].(string))
		parents = append(parents, pathTail(f["path"], 2)[0])
	}
	sort.Strings(pairs)
	sort.Strings(parents)
	if !slices.Equal(pairs, []string{"proxyExt|control:NullProxyExt", "proxyExt|control:NullProxyExt"}) ||
		!slices.Equal(parents, []string{"En", "Sp"}) {
		t.Errorf("frozen_children_not_restored = %v", frozen)
	}
}

func TestDescendantsOfASkippedFrozenChildAreSkippedToo(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	d.Add(d.node(nn, "Sp", "proxyExt").Handle, "Inner", "")
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "partial" { // Inner is gone after the rollback: never "verified" (issue #179 R4-002)
		t.Errorf("back = %v", back)
	}
	if destructiveContainsAny(names(adds(sent())), "Inner") {
		t.Errorf("Inner was re-added: %v", names(adds(sent())))
	}
	found := false
	for _, f := range objects(back["frozen_children_not_restored"]) {
		found = found || (f["n"] == "Inner" && f["type"] == "kitControl:NumericConst")
	}
	if !found {
		t.Errorf("frozen_children_not_restored = %v", back["frozen_children_not_restored"])
	}
	lost := map[string]map[string]any{}
	for _, f := range objects(back["frozen_config_not_restored"]) {
		lost[pathTail(f["path"], 1)[0]] = f
	}
	if len(lost) != 1 || lost["Inner"] == nil || lost["Inner"]["missing"] != true {
		t.Errorf("frozen_config_not_restored = %v", back["frozen_config_not_restored"])
	}
}

func TestARollbackWithoutFrozenChildrenReportsNone(t *testing.T) {
	d := destructiveRig(t, nil)
	_, removed := d.removeGroup()
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" { // a real rollback, so the absence is not vacuous
		t.Fatalf("back = %v", back)
	}
	if _, ok := back["frozen_children_not_restored"]; ok {
		t.Errorf("back = %v", back)
	}
}

func TestFrozenChildConfigNotRestoredDowngradesTheVerdictToPartial(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "partial" {
		t.Errorf("back = %v", back)
	}
	lost := objects(back["frozen_config_not_restored"])
	if len(lost) != 1 || !slices.Equal(pathTail(lost[0]["path"], 2), []string{"Sp", "proxyExt"}) {
		t.Fatalf("frozen_config_not_restored = %v", lost)
	}
	if !same(lost[0]["slots"], []any{"tuningPolicyName"}) || lost[0]["missing"] != false ||
		!same(Dig(lost[0], "captured", "s"), []any{map[string]any{"nm": "p", "t": "baja:String", "n": "tuningPolicyName", "v": "Fast"}}) {
		t.Errorf("lost = %v", lost[0])
	}
	if v := d.batch(back["batch_id"])["verdict"]; v != "partial" {
		t.Errorf("journaled verdict = %v", v)
	}
}

func TestAFrozenChildOfAnotherTypeIsReportedAsNotRestored(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	ext := d.node(nn, "Sp", "proxyExt")
	d.Station.Do(func(*boxstation.Station) { ext.Type = "bacnet:BacnetNumericProxyExt" })
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "partial" {
		t.Errorf("back = %v", back)
	}
	lost := objects(back["frozen_config_not_restored"])
	if len(lost) != 1 || lost[0]["type"] != "bacnet:BacnetNumericProxyExt" || lost[0]["live_type"] != "control:NullProxyExt" {
		t.Errorf("frozen_config_not_restored = %v", lost)
	}
}

// The fresh frozen child already holds the captured config: nothing was lost.
func TestAFrozenChildWhoseConfigMatchesKeepsTheVerdictVerified(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.writableGroup()
	d.configureProxyExt(nn)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) {
		if ext := n.Child("proxyExt"); n.Name == "Sp" && ext != nil { // model only: no BOX call in a hook
			ext.Children = append(ext.Children, &boxstation.Node{Name: "tuningPolicyName", Type: "baja:String", Value: "Fast"})
		}
	})
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	if _, ok := back["frozen_config_not_restored"]; ok {
		t.Errorf("back = %v", back)
	}
}
