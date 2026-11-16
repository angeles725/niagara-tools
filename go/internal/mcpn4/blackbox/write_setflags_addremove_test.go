package blackbox

// n4_set_flags add/remove: change some letters of a slot's flags without restating the whole set
// (bench finding: the full-set form dropped a letter the caller did not know was there).

import (
	"reflect"
	"strings"
	"testing"
)

func (f *flagsRig) edit(extra Args) Args {
	out := Args{"ord": f.ord, "slot": "note"}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

func TestSetFlagsAddKeepsTheLettersTheSlotHeld(t *testing.T) {
	f := newFlagsRig(t, "h")
	plan := planOf(f.Srv.Dry("n4_set_flags", f.edit(Args{"add": "o"})))
	wantNotes := []any{"flags of slot 'note' on " + f.ord + `: current "h" (h hidden) -> requested "ho" (h hidden, o operator)`}
	if !same(plan["ops"], []any{f.op("ho")}) || !same(plan["inverse"], []any{f.op("h")}) || !reflect.DeepEqual(plan["notes"], wantNotes) {
		t.Errorf("plan = %v", plan)
	}
	out := f.Srv.RunWrite("n4_set_flags", f.edit(Args{"add": "o"}))
	if out["verdict"] != "verified" || f.flags() != "ho" {
		t.Errorf("out = %v, flags %q", out, f.flags())
	}
}

func TestSetFlagsRemoveDropsOnlyTheNamedLetters(t *testing.T) {
	f := newFlagsRig(t, "ho")
	out := f.Srv.RunWrite("n4_set_flags", f.edit(Args{"remove": "o"}))
	if out["verdict"] != "verified" || f.flags() != "h" || !same(out["inverse"], []any{f.op("ho")}) {
		t.Errorf("out = %v, flags %q", out, f.flags())
	}
}

func TestSetFlagsAddAndRemoveInOneCall(t *testing.T) {
	f := newFlagsRig(t, "h")
	plan := planOf(f.Srv.Dry("n4_set_flags", f.edit(Args{"add": "os", "remove": "h"})))
	if !same(plan["ops"], []any{f.op("so")}) && !same(plan["ops"], []any{f.op("os")}) {
		t.Errorf("ops = %v", plan["ops"])
	}
}

func TestSetFlagsAddRemoveRefusals(t *testing.T) {
	f := newFlagsRig(t, "h")
	for _, tc := range []struct {
		args Args
		want string
	}{
		{Args{"flags": "ho", "add": "o"}, "either flags or add/remove, not both"},
		{Args{}, "give flags (the whole new set) or add and/or remove"},
		{Args{"add": "o", "remove": "o"}, "flag letter 'o' is both in add and in remove"},
		{Args{"add": "z"}, "unknown flag letter 'z' in add 'z'"},
		{Args{"remove": "oo"}, "flag letter 'o' is repeated in remove 'oo'"},
		{Args{"add": "h"}, "already has flags"},
		{Args{"add": ""}, "add and remove name no letter"},
	} {
		text := f.Srv.Err("n4_set_flags", f.edit(tc.args))
		if !strings.Contains(text, tc.want) {
			t.Errorf("%v: refusal = %s, want %q", tc.args, text, tc.want)
		}
	}
	f.untouched(t, "h")
}
