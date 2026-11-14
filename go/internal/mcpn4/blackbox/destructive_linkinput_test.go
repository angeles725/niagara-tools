package blackbox

// Null types and link-driven inputs (port of test_tools_destructive.TestRollback: live finding
// 3, and issue #179 R3-002 / B1200-G2).

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// priorityFacet is the runtime facet a live station puts on a link-driven input's status.
const priorityFacet = "0;activeLevel=e:17@control:PriorityLevel"

// liveInput adds a Status input to the component at path as the live station loads it: its
// frozen value and status children carry no type.
func (d *dRig) liveInput(path []string, name, value, status string) {
	comp := d.node(path...)
	d.Station.Do(func(*boxstation.Station) {
		comp.Children = append(comp.Children, &boxstation.Node{Name: name, Type: "baja:StatusNumeric",
			Children: []*boxstation.Node{{Name: "value", Value: value}, {Name: "status", Value: status}}})
	})
}

// linkedGroup is a group whose Tgt holds the link-driven in10 (4.5) and the plain inB (25.0).
func (d *dRig) linkedGroup() string {
	d.T.Helper()
	nn, _ := d.group("Grp")
	d.liveInput([]string{nn, "Tgt"}, "in10", "4.5", priorityFacet)
	d.liveInput([]string{nn, "Tgt"}, "inB", "25.0", priorityFacet)
	return nn
}

// statusValues maps the children of a captured Status body to their values.
func statusValues(body any) map[string]any {
	out := map[string]any{}
	for _, c := range objects(Dig(body, "s")) {
		n, _ := c["n"].(string)
		out[n] = c["v"]
	}
	return out
}

// entries are the values under key of the inverse records that carry it.
func entries(inverse any, key string) []map[string]any {
	out := []map[string]any{}
	for _, e := range objects(inverse) {
		if m, ok := e[key].(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func TestALinkedGroupRollsBackWithoutNullTypesOrLinkDrivenInputs(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.linkedGroup()
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	sent := d.f.record()
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("back = %v", back)
	}
	var bodies []any
	for _, op := range sent() {
		bodies = append(bodies, op["b"])
	}
	if raw, _ := json.Marshal(bodies); len(bodies) == 0 || strings.Contains(string(raw), `"t":null`) {
		t.Errorf("sent bodies = %s", raw)
	}
	tgt := addOf(sent(), "Tgt")
	slots := byName(Dig(tgt, "b", "s"))
	if _, ok := slots["in10"]; ok { // link target: the relink re-establishes it
		t.Errorf("Tgt add carries in10: %v", tgt)
	}
	inB, ok := slots["inB"] // plain configured input is kept
	if !ok {
		t.Fatalf("Tgt add lacks inB: %v", tgt)
	}
	kids := byName(inB["s"])
	if kids["status"]["v"] != "0" { // bits kept, facet dropped
		t.Errorf("inB status = %v", kids["status"])
	}
	for _, c := range objects(inB["s"]) {
		if _, typed := c["t"]; !typed && c["n"] != "value" && c["n"] != "status" {
			t.Errorf("inB child without a type: %v", c)
		}
	}
	if _, typed := kids["value"]["t"]; typed {
		t.Errorf("inB value carries a type: %v", kids["value"])
	}
	if !same(back["relinks"], map[string]any{"restored": 1, "skipped": 0}) {
		t.Errorf("relinks = %v", back["relinks"])
	}
	if v := d.value(d.node(nn).Handle, "Tgt", "Link", "targetSlotName"); v != "in10" {
		t.Errorf("Tgt link target slot = %v", v)
	}
}

func TestTheRemovePlanRecordsTheValueOfALinkDrivenInput(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.linkedGroup()
	reply := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	plan := planOf(reply)
	relinks := entries(plan["inverse"], "relink")
	if len(relinks) != 1 || relinks[0]["target_slot"] != "in10" {
		t.Fatalf("relinks = %v", relinks)
	}
	if _, ok := relinks[0]["prior"]; ok { // volatile: never part of the hashed plan (PR #187 review R3-001)
		t.Errorf("the hashed relink carries prior: %v", relinks[0])
	}
	if len(noteWith(plan["notes"], "captured when the confirmed remove runs")) == 0 {
		t.Errorf("notes = %v", plan["notes"])
	}
	seen := objects(reply["link_input_values"]) // the operator's preview, outside the hash
	if len(seen) != 1 {
		t.Fatalf("link_input_values = %v", reply["link_input_values"])
	}
	if seen[0]["target_path"] != "Tgt" || seen[0]["target_slot"] != "in10" || Dig(seen[0], "value", "t") != "baja:StatusNumeric" {
		t.Errorf("link_input_values = %v", seen)
	}
	if got := statusValues(seen[0]["value"]); !same(got, map[string]any{"value": "4.5", "status": "0"}) { // facets dropped, bits kept
		t.Errorf("value = %v", got)
	}
}

// The link source keeps propagating between dry run and confirm (PR #187 blocking review R3-001).
func TestALinkDrivenValueThatChangesAfterTheDryRunKeepsTheTokenValid(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.linkedGroup()
	plan := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	value := d.node(nn, "Tgt", "in10", "value")
	d.Station.Do(func(*boxstation.Station) { value.Value = "7.25" })
	removed := d.Srv.OK("n4_remove_component", Execute(removeArgs(nn), plan["confirmation_token"]))
	if slices.Contains(d.children(), nn) {
		t.Errorf("children = %v", d.children())
	}
	relinks := entries(d.batch(removed["batch_id"])["inverse"], "relink")
	if len(relinks) != 1 || !same(statusValues(relinks[0]["prior"]), map[string]any{"value": "7.25", "status": "0"}) { // the value seen at execution
		t.Errorf("relinks = %v", relinks)
	}
}

func TestASkippedRelinkReportsTheCapturedInputValue(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.linkedGroup()
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	d.afterAdd(func(st *boxstation.Station, n *boxstation.Node) {
		if n.Name == "Src" {
			removeChild(st.ByHandle("3").Child(nn), "Src")
		}
	})
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "mismatch" {
		t.Errorf("back = %v", back)
	}
	lost := objects(back["link_inputs_not_restored"])
	if len(lost) != 1 {
		t.Fatalf("link_inputs_not_restored = %v", back["link_inputs_not_restored"])
	}
	if target, _ := lost[0]["target"].(string); !strings.HasSuffix(target, "/"+nn+"/Tgt") ||
		lost[0]["target_slot"] != "in10" || lost[0]["source_slot"] != "out" || lost[0]["reason"] != "relink skipped" {
		t.Errorf("lost = %v", lost[0])
	}
	if got := statusValues(lost[0]["value"]); !same(got, map[string]any{"value": "4.5", "status": "0"}) {
		t.Errorf("value = %v", got)
	}
}

func TestAnInputLinkedFromOutsideTheSubtreeIsReportedNotLost(t *testing.T) {
	d := destructiveRig(t, nil)
	nn := d.linkedGroup()
	other := d.AddWithOut("Other", 1.0)
	d.liveInput([]string{nn, "Tgt"}, "in11", "1.0", priorityFacet)
	d.Link(other, "out", Component{"Tgt", d.node(nn, "Tgt").Handle}, "in11")
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
	unlinked := entries(d.batch(removed["batch_id"])["inverse"], "unlinked_input")
	if len(unlinked) != 1 || unlinked[0]["target_path"] != "Tgt" || unlinked[0]["target_slot"] != "in11" {
		t.Errorf("unlinked inputs = %v", unlinked)
	}
	back := d.rollback(removed["batch_id"])
	if back["verdict"] != "partial" {
		t.Errorf("back = %v", back)
	}
	lost := objects(back["link_inputs_not_restored"])
	if len(lost) != 1 {
		t.Fatalf("link_inputs_not_restored = %v", back["link_inputs_not_restored"])
	}
	if lost[0]["target_slot"] != "in11" || lost[0]["reason"] != "link source outside the removed subtree" {
		t.Errorf("lost = %v", lost[0])
	}
	if got := statusValues(lost[0]["value"]); !same(got, map[string]any{"value": "1.0", "status": "0"}) {
		t.Errorf("value = %v", got)
	}
}
