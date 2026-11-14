package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// Flags an integrator sets (Workbench Config Flags) are configuration: the platform's own clone
// skips a dynamic slot only for R (ComponentSlotMap.newCopy) and a frozen one only for its
// declared d (ComplexSlotMap.newCopy). A transient slot is never persisted, so it holds nothing.

// (k) A dynamic child flagged readonly is copied whole, and the add op carries its flags.
func TestACopyKeepsAReadonlyDynamicChildWithItsFlags(t *testing.T) {
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "Cfg", "t", "baja:Folder", "f", "r", "s", []any{
		obj("nm", "p", "n", "limit", "t", "baja:Double", "v", "5.0", "f", "rh")}))
	_, op, notes := copyPlanOf(t, tree)
	want := `{"nm": "p", "t": "baja:Folder", "n": "Cfg", "s": [{"nm": "p", "t": "baja:Double", "n": "limit", "v": "5.0", "f": "rh"}], "f": "r"}`
	if !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
	if strings.Contains(notes, "Cfg") {
		t.Errorf("notes = %s", notes)
	}
}

// (l) A frozen configuration slot an integrator made readonly is copied with its flags.
func TestACopyKeepsAFrozenConfigSlotMadeReadonly(t *testing.T) {
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "enabled", "t", "baja:Boolean", "v", "false", "f", "r"))
	_, op, _ := copyPlanOf(t, tree)
	if want := `{"nm": "p", "t": "baja:Boolean", "n": "enabled", "v": "false", "f": "r"}`; !strings.Contains(op, want) {
		t.Errorf("op %s\n lacks %s", op, want)
	}
}

// (m) A dynamic slot flagged remove-on-clone is left out and listed.
func TestACopyLeavesOutADynamicRemoveOnCloneSlot(t *testing.T) {
	tree := gwTree()
	addKids(tree, obj("nm", "p", "n", "Tag", "t", "baja:String", "v", "x", "f", "R"))
	_, op, notes := copyPlanOf(t, tree)
	if strings.Contains(op, `"Tag"`) || !strings.Contains(notes, "Tag (R)") {
		t.Errorf("op = %s\nnotes = %s", op, notes)
	}
}

// (n) A frozen slot an integrator made transient is not persisted: left out and listed.
func TestACopyLeavesOutAFrozenSlotMadeTransient(t *testing.T) {
	tree := gwTree()
	child(tree, "deviceAddress").Set("f", "t")
	_, op, notes := copyPlanOf(t, tree)
	if strings.Contains(op, `"deviceAddress"`) || !strings.Contains(notes, "deviceAddress (t)") {
		t.Errorf("op = %s\nnotes = %s", op, notes)
	}
}

// A child that is not an object is a malformed reply: refused, naming the node that holds it.
func TestACopyRefusesAMalformedChild(t *testing.T) {
	tree := gwTree()
	addKids(under(tree, "points/Setpoint"), "junk")
	_, err := writeops.CopyComponent.Plan(copyClient(t, tree), copyArgs("Gw3"))
	if err == nil || err.Error() != "malformed station reply: a child of 'points/Setpoint' is not an object" {
		t.Errorf("err = %v", err)
	}
}

// The read-back compares flags too: a copy whose slot flags differ from the plan is a mismatch.
func TestACopyReadBackComparesSlotFlags(t *testing.T) {
	flagged := gwTree()
	child(flagged, "deviceAddress").Set("f", "r")
	observed, verdict := readBackCopy(t, "Gw3", flagged)
	if verdict != "mismatch" || !strings.Contains(observed, `"differences": ["deviceAddress"]`) {
		t.Errorf("flags changed: %s %s", verdict, observed)
	}
}
