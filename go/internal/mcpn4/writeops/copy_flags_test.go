package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// child is the loaded child of node named n (nil when absent).
func child(node *pyjson.Object, n string) *pyjson.Object {
	kids, _ := node.Get("s")
	list, _ := kids.([]any)
	for _, k := range list {
		if o := k.(*pyjson.Object); get(o, "n") == n {
			return o
		}
	}
	return nil
}

// under is the node at path below root ("points/Setpoint").
func under(root *pyjson.Object, path string) *pyjson.Object {
	for _, n := range strings.Split(path, "/") {
		root = child(root, n)
	}
	return root
}

// addKids appends kids to node's children.
func addKids(node *pyjson.Object, kids ...any) {
	old, _ := node.Get("s")
	list, _ := old.([]any)
	node.Set("s", append(list, kids...))
}

func copyPlanOf(t *testing.T, tree *pyjson.Object) (*plan.Planned, string, string) {
	t.Helper()
	p, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err != nil {
		t.Fatal(err)
	}
	return p, js(t, p.Ops[0]), js(t, p.Notes)
}

// Every stripped node is listed with its flags, in tree order, in one note.
func TestACopyListsEveryNodeItLeavesOutByItsFlags(t *testing.T) {
	_, _, notes := copyPlanOf(t, gwTree())
	want := `["Not copied (flags r readonly, t transient, d default-on-clone, R remove-on-clone): ` +
		`status (rtsd), health (rd), points/Setpoint/out (rtso), points/Setpoint/proxyExt/status (rt), ` +
		`points/Setpoint/proxyExt/readValue (rt)", ` +
		`"The station may rename on collision; the real inverse uses the assigned name."]`
	if notes != want {
		t.Errorf("notes\n got  %s\n want %s", notes, want)
	}
}

// (a) Children merely named like a runtime slot are configuration and are copied whole.
func TestACopyKeepsChildrenNamedStatusOrHealth(t *testing.T) {
	tree := gwTree()
	addKids(under(tree, "points"), obj("nm", "p", "n", "status", "t", "baja:Folder", "s", []any{
		obj("nm", "p", "n", "health", "t", "baja:Folder", "s", []any{
			obj("nm", "p", "n", "note", "t", "baja:String", "v", "keep")})}))
	_, op, _ := copyPlanOf(t, tree)
	want := `{"nm": "p", "t": "baja:Folder", "n": "status", "s": [{"nm": "p", "t": "baja:Folder", "n": "health", ` +
		`"s": [{"nm": "p", "t": "baja:String", "n": "note", "v": "keep"}]}]}`
	if !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
}

// (b) A fallback's status is part of the configured value (e.g. null) and is kept.
func TestACopyKeepsAFallbackStatus(t *testing.T) {
	tree := gwTree()
	addKids(under(tree, "points/Setpoint/fallback"), obj("nm", "p", "n", "status", "t", "baja:Status", "v", "8"))
	_, op, _ := copyPlanOf(t, tree)
	want := `{"nm": "p", "t": "baja:StatusNumeric", "n": "fallback", "s": [{"nm": "p", "n": "value", "v": "200.0"}, ` +
		`{"nm": "p", "t": "baja:Status", "n": "status", "v": "8"}]}`
	if !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
}

// (c) A frozen slot arrives without "f" at its default flags: the type contract supplies them.
func TestACopyLeavesOutAFrozenSlotItsContractDeclaresTransient(t *testing.T) {
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "Poll", "t", "driver:BasicPollScheduler", "s", []any{
		obj("nm", "p", "n", "averagePoll", "t", "baja:String", "v", "12ms"),
		obj("nm", "p", "n", "pollEnabled", "t", "baja:Boolean", "v", "false")}))
	_, op, notes := copyPlanOf(t, tree)
	if strings.Contains(op, "averagePoll") || !strings.Contains(op, "pollEnabled") {
		t.Errorf("op = %s", op)
	}
	if !strings.Contains(notes, "Poll/averagePoll (rt)") {
		t.Errorf("notes = %s", notes)
	}
}

// (d) The declared flags decide a frozen slot: an integrator who clears a runtime slot's instance
// flags does not make it configuration. (Before #618 review this test pinned the opposite rule,
// instance flags first; flags an integrator sets are now configuration, see copy_integrator_test.go.)
func TestACopyUsesInstanceFlagsOverTheContract(t *testing.T) {
	tree := gwTree()
	child(tree, "health").Set("f", "")
	_, op, notes := copyPlanOf(t, tree)
	if strings.Contains(op, `"health"`) {
		t.Errorf("op = %s", op)
	}
	if !strings.Contains(notes, "health (rd)") {
		t.Errorf("notes = %s", notes)
	}
}

// (e) A readonly-only slot off its declared default is called out: its value is not copied.
func TestACopyCallsOutAReadonlySlotOffItsDefault(t *testing.T) {
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "AlarmExt", "t", "alarm:AlarmSourceExt", "s", []any{
		obj("nm", "p", "n", "toOffnormalTimes", "t", "alarm:AlarmTimestamps", "v", "x"),
		obj("nm", "p", "n", "toFaultTimes", "t", "alarm:AlarmTimestamps", "v", "null"),
		obj("nm", "p", "n", "ackedTransitions", "t", "alarm:AlarmTransitionBits", "v", "0")}))
	_, _, notes := copyPlanOf(t, tree)
	if !strings.Contains(notes, `"readonly slot holds a non-default value and is not copied: AlarmExt/toOffnormalTimes"`) {
		t.Errorf("notes = %s", notes)
	}
	if strings.Contains(notes, "copied: AlarmExt/toFaultTimes") || strings.Contains(notes, "copied: AlarmExt/ackedTransitions") {
		t.Errorf("a default or default-on-clone slot was called out: %s", notes)
	}
}

// (f) Without the registry channel, or for a type the station does not describe, nothing is planned.
func TestACopyRefusesWhenItCannotReadTheTypes(t *testing.T) {
	c := copyClient(t, gwTree())
	_, err := writeops.CopyComponent.Plan(&c.loads, copyArgs("Gw3"))
	if err == nil || err.Error() != "a copy needs the station's type definitions (registry channel) to leave "+
		"runtime slots out; this client cannot read them" {
		t.Errorf("no registry: err = %v", err)
	}
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "W", "t", "acme:Widget", "s", []any{obj("nm", "p", "n", "x", "t", "baja:String", "v", "1")}))
	_, err = writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err == nil || err.Error() != "the station did not describe type 'acme:Widget'; a copy cannot tell its "+
		"runtime slots from its configuration" {
		t.Errorf("unknown type: err = %v", err)
	}
}

// One registry call per plan covers every type the copy evaluates.
func TestACopyLoadsItsTypesInOneRegistryCall(t *testing.T) {
	c := copyClient(t, gwTree())
	if _, err := writeops.CopyComponent.Plan(c, copyArgs("Gw3")); err != nil {
		t.Fatal(err)
	}
	if len(c.calls) != 1 {
		t.Errorf("calls = %v, want one", c.calls)
	}
}
