package blackbox

// n4_set_slot (port of test_tools_write TestSetSlot).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// calcRig is TestSetSlot's setUp: a verified session and Calc, whose out holds 1.0.
func calcRig(t *testing.T) (*Rig, Component, string) {
	r := verifiedRig(t)
	c := r.AddWithOut("Calc", 1.0)
	return r, c, Folder + "/" + c.Name
}

func setOp(h, slot string, body map[string]any) []any {
	return []any{map[string]any{"nm": "s", "h": h, "n": slot, "b": body}}
}

func TestStatusNumericHappyPathAndInverseRestoresPrevious(t *testing.T) {
	r, c, ord := calcRig(t)
	args := Args{"ord": ord, "slot": "out", "value": 5.5, "value_type": "baja:StatusNumeric"}
	plan := r.Srv.Dry("n4_set_slot", args)
	if !reflect.DeepEqual(Dig(plan, "plan", "ops"), setOp(c.Handle, "out", StatusNumeric(5.5, "0"))) ||
		!reflect.DeepEqual(Dig(plan, "plan", "inverse"), setOp(c.Handle, "out", StatusNumeric(1.0, "0"))) {
		t.Errorf("plan = %v", plan["plan"])
	}
	out := r.Srv.RunWrite("n4_set_slot", args)
	want := map[string]any{"value": 5.5, "status": "0"}
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["requested"], want) || !reflect.DeepEqual(out["observed"], want) {
		t.Errorf("out = %v", out)
	}
}

func TestStatusBooleanIsWrittenWhole(t *testing.T) {
	r, c, ord := calcRig(t)
	r.Set(c.Handle, "flag", StatusBoolean(false, "0"))
	out := r.Srv.RunWrite("n4_set_slot", Args{"ord": ord, "slot": "flag", "value": true, "value_type": "baja:StatusBoolean"})
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["observed"], map[string]any{"value": true, "status": "0"}) {
		t.Errorf("out = %v", out)
	}
}

func TestPlainTypesHappyPath(t *testing.T) {
	r, c, ord := calcRig(t)
	for _, tc := range []struct {
		slot, typ, first string
		new              any
	}{{"d", "baja:Double", "2.0", 3.5}, {"b", "baja:Boolean", "true", false}, {"s", "baja:String", "old", "new"}} {
		r.Set(c.Handle, tc.slot, map[string]any{"nm": "p", "t": tc.typ, "v": tc.first})
		out := r.Srv.RunWrite("n4_set_slot", Args{"ord": ord, "slot": tc.slot, "value": tc.new, "value_type": tc.typ})
		if out["verdict"] != "verified" || out["observed"] != tc.new {
			t.Errorf("%s: out = %v", tc.slot, out)
		}
	}
}

func TestValueOrStatusChildPathIsRefusedWithTheReason(t *testing.T) {
	r, c, ord := calcRig(t)
	for _, slot := range []string{"out/value", "out/status"} {
		text := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": slot, "value": 2.0, "value_type": "baja:Double"})
		if !strings.Contains(text, "whole") || !strings.Contains(text, "status") {
			t.Errorf("%s: refusal = %s", slot, text)
		}
	}
	if got := nodeValue(r, c.Handle, "out", "value"); got != "1.0" {
		t.Errorf("out/value = %v", got)
	}
}

// F13 (audit 2026-10-03): untested branch.
func TestAStatusForAPlainValueTypeIsRefused(t *testing.T) {
	r, c, ord := calcRig(t)
	r.Set(c.Handle, "d", map[string]any{"nm": "p", "t": "baja:Double", "v": "2.0"})
	text := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": "d", "value": 3.0, "value_type": "baja:Double", "status": "0"})
	if !strings.Contains(text, "status only applies") {
		t.Errorf("refusal = %s", text)
	}
}

func TestBadValuesTypesAndStatusAreRefused(t *testing.T) {
	r, _, ord := calcRig(t)
	base := Args{"ord": ord, "slot": "out"}
	for _, bad := range []Args{{"value": true, "value_type": "baja:Double"},
		{"value": "x", "value_type": "baja:Boolean"},
		{"value": 1, "value_type": "baja:String"},
		{"value": 1.0, "value_type": "baja:Int"},
		{"value": 1.0, "value_type": "baja:StatusNumeric", "status": "zz"}} {
		r.Srv.Err("n4_set_slot", withArgs(base, bad))
	}
	nan := callNaN(t, r.Srv, "n4_set_slot", withArgs(base, Args{"value_type": "baja:Double"}), "value")
	if nan["isError"] != true {
		t.Errorf("NaN: %v", nan)
	}
}

func TestSlotAtItsTypeDefaultIsNotListedByTheStationSoItIsRefused(t *testing.T) {
	r, c, ord := calcRig(t)
	r.Set(c.Handle, "zero", map[string]any{"nm": "p", "t": "baja:Double", "v": "0.0"}) // default values are omitted
	text := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": "zero", "value": 1.0, "value_type": "baja:Double"})
	// F10 (audit 2026-10-03): the refusal explains why and names the routes.
	for _, needle := range []string{"not found", "type default", "n4_read_slots", "Workbench"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
}

// F3 (audit 2026-10-03): `grp/sp` needs its value/status children loaded, else the previous
// value (and the journaled inverse) silently became the type default.
func TestANestedStatusSlotPlansItsRealPreviousValue(t *testing.T) {
	r, c, ord := calcRig(t)
	sp := StatusNumeric(7.5, "0")
	sp["n"] = "sp"
	r.Set(c.Handle, "grp", map[string]any{"nm": "p", "t": "baja:Struct", "s": []any{sp}})
	args := Args{"ord": ord, "slot": "grp/sp", "value": 9.0, "value_type": "baja:StatusNumeric"}
	plan := r.Srv.Dry("n4_set_slot", args)
	if got := Dig(plan, "plan", "inverse"); !reflect.DeepEqual(got, setOp(c.Handle, "grp/sp", StatusNumeric(7.5, "0"))) {
		t.Errorf("inverse = %v", got)
	}
	out := r.Srv.OK("n4_set_slot", Execute(args, plan["confirmation_token"]))
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["observed"], map[string]any{"value": 9.0, "status": "0"}) {
		t.Errorf("out = %v", out)
	}
}

func TestUnknownSlotAndTypeMismatchWithTheSlotAreRefused(t *testing.T) {
	r, _, ord := calcRig(t)
	if text := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": "nope", "value": 1.0, "value_type": "baja:Double"}); !strings.Contains(text, "not found") {
		t.Errorf("unknown slot: %s", text)
	}
	if text := r.Srv.Err("n4_set_slot", Args{"ord": ord, "slot": "out", "value": true, "value_type": "baja:StatusBoolean"}); !strings.Contains(text, "baja:StatusNumeric") {
		t.Errorf("type mismatch: %s", text)
	}
}

func TestSetSlotReadBackMismatchIsReportedNotRaised(t *testing.T) {
	r, c, ord := calcRig(t)
	r.Station.SetFaults(boxstation.Faults{OnSync: func(st *boxstation.Station, _ map[string]any) {
		st.ByHandle(c.Handle).Child("out").Child("value").Value = "99.0"
	}})
	out := r.Srv.RunWrite("n4_set_slot", Args{"ord": ord, "slot": "out", "value": 5.0, "value_type": "baja:StatusNumeric"})
	if out["verdict"] != "mismatch" || Dig(out, "observed", "value") != 99.0 {
		t.Errorf("out = %v", out)
	}
	if got := phases(r); !reflect.DeepEqual(got, []any{"intent", "result"}) { // still journaled
		t.Errorf("journal phases = %v", got)
	}
}
