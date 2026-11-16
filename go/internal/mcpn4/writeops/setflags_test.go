package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

// flagsRigArgs are n4_set_flags arguments on the history extension of the typed rig (handle 42,
// type history:NumericIntervalHistoryExt; its contract declares enabled "o", status "rt",
// historyName and the action enable without flags).
func flagsRigArgs(slot, flags string) *pyjson.Object {
	return obj("ord", ext, "slot", slot, "flags", flags)
}

// flagsRigNote is the plan note that shows the current and the requested flags.
func flagsRigNote(slot, current, requested string) string {
	return "flags of slot '" + slot + "' on " + ext + ": current " + current + " -> requested " + requested
}

func flagsRigPlan(t *testing.T, c plan.Client, slot, flags string) *plan.Planned {
	t.Helper()
	p, err := writeops.SetFlags.Plan(c, flagsRigArgs(slot, flags))
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	return p
}

// The op is the station's SetFlagsOp ({nm: f, h, n, f}); the inverse restores the flags the
// slot holds, read from the loaded slot or, for a frozen slot at its declared flags, from the
// type contract; the plan names both sets.
func TestSetFlagsPlansTheOpTheExactInverseAndNamesBothSets(t *testing.T) {
	for _, tc := range []struct {
		name, slot, flags string
		node              *pyjson.Object // the loaded slot, nil when the station omits it
		op, inverse, note string
	}{
		{"a dynamic slot with flags", "note", "ho", obj("nm", "p", "t", "baja:String", "v", "x", "f", "h"),
			`{"nm": "f", "h": "42", "n": "note", "f": "ho"}`, `{"nm": "f", "h": "42", "n": "note", "f": "h"}`,
			flagsRigNote("note", `"h" (h hidden)`, `"ho" (h hidden, o operator)`)},
		{"the inverse is the station's own string, not a reordered one", "note", "s", obj("nm", "p", "t", "baja:String", "v", "x", "f", "oh"),
			`{"nm": "f", "h": "42", "n": "note", "f": "s"}`, `{"nm": "f", "h": "42", "n": "note", "f": "oh"}`,
			flagsRigNote("note", `"oh" (o operator, h hidden)`, `"s" (s summary)`)},
		{"a dynamic slot without flags", "note", "s", obj("nm", "p", "t", "baja:String", "v", "x"),
			`{"nm": "f", "h": "42", "n": "note", "f": "s"}`, `{"nm": "f", "h": "42", "n": "note", "f": ""}`,
			flagsRigNote("note", `"" (none)`, `"s" (s summary)`)},
		{"a frozen slot the station omits: its declared flags", "enabled", "", nil,
			`{"nm": "f", "h": "42", "n": "enabled", "f": ""}`, `{"nm": "f", "h": "42", "n": "enabled", "f": "o"}`,
			flagsRigNote("enabled", `"o" (o operator)`, `"" (none)`)},
		{"a frozen slot listed without flags: its declared flags", "status", "hrt", obj("t", "baja:Status", "v", "0"),
			`{"nm": "f", "h": "42", "n": "status", "f": "rth"}`, `{"nm": "f", "h": "42", "n": "status", "f": "rt"}`,
			flagsRigNote("status", `"rt" (r readonly, t transient)`, `"rth" (r readonly, t transient, h hidden)`)},
		{"a frozen slot whose instance flags were cleared", "enabled", "oh", obj("t", "baja:Boolean", "v", "true", "f", ""),
			`{"nm": "f", "h": "42", "n": "enabled", "f": "ho"}`, `{"nm": "f", "h": "42", "n": "enabled", "f": ""}`,
			flagsRigNote("enabled", `"" (none)`, `"ho" (h hidden, o operator)`)},
		{"a frozen action the station omits", "enable", "h", nil,
			`{"nm": "f", "h": "42", "n": "enable", "f": "h"}`, `{"nm": "f", "h": "42", "n": "enable", "f": ""}`,
			flagsRigNote("enable", `"" (none)`, `"h" (h hidden)`)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			nodes := extNodes()
			if tc.node != nil {
				nodes.Set(tc.slot, tc.node)
			}
			p := flagsRigPlan(t, newTyped(t, nodes), tc.slot, tc.flags)
			if len(p.Ops) != 1 || js(t, p.Ops[0]) != tc.op {
				t.Errorf("ops\n got  %s\n want %s", js(t, p.Ops), tc.op)
			}
			if len(p.Inverse) != 1 || js(t, p.Inverse[0]) != tc.inverse {
				t.Errorf("inverse\n got  %s\n want %s", js(t, p.Inverse), tc.inverse)
			}
			if got, want := js(t, p.Notes), js(t, []any{tc.note}); got != want {
				t.Errorf("notes\n got  %s\n want %s", got, want)
			}
			if got := js(t, p.Targets); got != `{"`+ext+`": "42"}` { // what a rollback checks
				t.Errorf("targets = %s", got)
			}
		})
	}
}

// The read-back loads the slot again: verified only when the flags it holds equal the requested
// set in any order, else mismatch with both; the op's reply alone decides nothing.
func TestSetFlagsReadBackComparesTheFlagsTheStationHolds(t *testing.T) {
	before := obj("nm", "p", "t", "baja:String", "v", "x", "f", "h")
	for _, tc := range []struct {
		name, slot, flags string
		after             *pyjson.Object // the slot after the op, nil when the station omits it
		want              string
	}{
		{"the requested set in the station's order", "note", "oh", obj("nm", "p", "t", "baja:String", "v", "x", "f", "ho"),
			`["ho", {"id": "f"}, "ho", "verified"]`},
		{"the requested set in another order", "note", "ho", obj("nm", "p", "t", "baja:String", "v", "x", "f", "oh"),
			`["ho", {"id": "f"}, "oh", "verified"]`},
		{"the station kept the previous flags", "note", "ho", before, `["ho", {"id": "f"}, "h", "mismatch"]`},
		{"the station holds one flag more", "note", "ho", obj("nm", "p", "t", "baja:String", "v", "x", "f", "hor"),
			`["ho", {"id": "f"}, "hor", "mismatch"]`},
		{"a dynamic slot cleared: listed without flags", "note", "", obj("nm", "p", "t", "baja:String", "v", "x"),
			`["", {"id": "f"}, "", "verified"]`},
		{"a dynamic slot asked to clear that still has none of it", "note", "", before, `["", {"id": "f"}, "h", "mismatch"]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := newTyped(t, extNodes("note", before))
			args := flagsRigArgs(tc.slot, tc.flags)
			p := flagsRigPlan(t, c, tc.slot, tc.flags)
			c.nodes = obj(ext, extNodes(tc.slot, tc.after))
			requested, accepted, observed, verdict, err := writeops.SetFlags.Readback(c, args, p, []any{obj("id", "f")}, p.Inverse)
			if err != nil {
				t.Fatalf("readback: %v", err)
			}
			if got := js(t, []any{requested, accepted, observed, verdict}); got != tc.want {
				t.Errorf("readback\n got  %s\n want %s", got, tc.want)
			}
		})
	}
}

// A frozen slot set back to its declared flags is listed without "f" again (or omitted): the
// read-back takes the declared flags from the contract, not "no flags".
func TestSetFlagsReadBackOfAFrozenSlotBackAtItsDeclaredFlagsIsVerified(t *testing.T) {
	c := newTyped(t, extNodes("enabled", obj("t", "baja:Boolean", "v", "true", "f", "")))
	args := flagsRigArgs("enabled", "o")
	p := flagsRigPlan(t, c, "enabled", "o")
	c.nodes = obj(ext, extNodes())
	requested, _, observed, verdict, err := writeops.SetFlags.Readback(c, args, p, []any{obj("id", "f")}, p.Inverse)
	if err != nil || requested != "o" || observed != "o" || verdict != "verified" {
		t.Errorf("readback = %v %v %v, err %v", requested, observed, verdict, err)
	}
}

// A slot that is gone at read-back is an error (the engine reports the batch as failed), never
// a verified or a mismatch.
func TestSetFlagsReadBackFailsWhenTheSlotIsGone(t *testing.T) {
	c := newTyped(t, extNodes("note", obj("nm", "p", "t", "baja:String", "v", "x")))
	args := flagsRigArgs("note", "h")
	p := flagsRigPlan(t, c, "note", "h")
	c.nodes = obj(ext, extNodes())
	_, _, _, verdict, err := writeops.SetFlags.Readback(c, args, p, []any{obj("id", "f")}, p.Inverse)
	if err == nil || verdict != "" {
		t.Errorf("verdict %q, err %v", verdict, err)
	}
}
