package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Bench retro 2026-11-14 row 10: a slot without "f" that no contract declares is taken for a
// dynamic slot with no flags. When the contract walk ended at a non-baja type the station
// answered with no contract ({}), that cannot be told from a frozen slot of an undescribed type: the plan says so.

func undescribedRig(types *pyjson.Object) *typed {
	plain := obj("nm", "p", "t", "baja:Boolean", "v", "true")
	return &typed{loads: loads{nodes: obj(ext, obj("", obj("h", "42", "t", "acme:Widget"), "x", plain))}, types: types}
}

func TestSetFlagsNamesTheUndescribedTypeAContractWalkEndedAt(t *testing.T) {
	c := undescribedRig(obj("acme:Widget", obj("p", "acme:Base"), "acme:Base", obj()))
	p := flagsRigPlan(t, c, "x", "h")
	if got := js(t, p.Inverse[0]); got != `{"nm": "f", "h": "42", "n": "x", "f": ""}` {
		t.Errorf("inverse = %s", got)
	}
	if len(p.Notes) != 2 || p.Notes[0] != flagsRigNote("x", `"" (none)`, `"h" (h hidden)`) {
		t.Fatalf("notes = %v", p.Notes)
	}
	note := pyjson.Str(p.Notes[1])
	for _, part := range []string{"'x'", "acme:Widget", "'acme:Base'", "no contract", "frozen slot", `inverse ""`} {
		if !strings.Contains(note, part) {
			t.Errorf("note lacks %q: %s", part, note)
		}
	}
}

func TestSetFlagsAddsNoNoteWhenTheContractWalkEndedAtTheRootOrAtDescribedTypes(t *testing.T) {
	for name, types := range map[string]*pyjson.Object{
		// the live station answers a common baja type that declares nothing with {}
		"the walk ends at a baja type answered with {}": obj("acme:Widget", obj("c", []any{}, "p", "baja:Component"),
			"baja:Component", obj()),
		"the walk ends at a type with slots and no supertype": obj("acme:Widget", obj("c", []any{}, "p", "baja:Component"),
			"baja:Component", obj("c", []any{})),
	} {
		p := flagsRigPlan(t, undescribedRig(types), "x", "h")
		if len(p.Notes) != 1 {
			t.Errorf("%s: notes = %v", name, p.Notes)
		}
	}
}
