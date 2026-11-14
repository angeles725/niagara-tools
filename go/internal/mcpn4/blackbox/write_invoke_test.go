package blackbox

// n4_invoke_action (port of test_tools_write TestInvokeAction).

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// spRig is TestInvokeAction's setUp: a verified session and Sp, a NumericWritable; extra runs
// after each accepted action.
func spRig(t *testing.T, extra func(st *boxstation.Station, action string, n *boxstation.Node)) (*Rig, Component, string, *invocations) {
	r := verifiedRig(t)
	c := r.Add("3", "Sp", "control:NumericWritable")
	return r, c, Folder + "/" + c.Name, recordInvocations(r, extra)
}

var setSeven = Args{"action": "set", "arg": 7.0, "arg_type": "baja:Double"}

func TestActionsOutsideTheAllowlistAreRefused(t *testing.T) {
	r, _, ord, inv := spRig(t, nil)
	for _, action := range []string{"save", "restart", "emergencyOverride", "emergencyAuto", "x"} {
		text := r.Srv.Err("n4_invoke_action", Args{"ord": ord, "action": action})
		if !strings.Contains(text, "not allowed in this version") ||
			strings.Contains(text, "n4_save_station") != (action == "save") {
			t.Errorf("%s: refusal = %s", action, text)
		}
	}
	if got := inv.get(); len(got) != 0 || r.Station.Saves() != 0 {
		t.Errorf("invoked %v, saves %d", got, r.Station.Saves())
	}
}

func TestSetHappyPathVerifiesFallbackAndInverseRestoresIt(t *testing.T) {
	r, c, ord, inv := spRig(t, nil)
	args := withArgs(setSeven, Args{"ord": ord})
	plan := r.Srv.Dry("n4_invoke_action", args)
	ops, _ := Dig(plan, "plan", "ops").([]any)
	wantArg := map[string]any{"h": c.Handle, "a": "set", "b": map[string]any{"nm": "p", "t": "baja:Double", "v": "7.0"}}
	if len(ops) == 0 || !reflect.DeepEqual(Dig(ops[0], "arg"), wantArg) ||
		!reflect.DeepEqual(Dig(plan, "plan", "inverse"), setOp(c.Handle, "fallback", StatusNumeric(0, "0"))) {
		t.Errorf("plan = %v", plan["plan"])
	}
	out := r.Srv.RunWrite("n4_invoke_action", args)
	if out["verdict"] != "verified" || !reflect.DeepEqual(out["observed"], map[string]any{"fallback": 7.0}) {
		t.Errorf("out = %v", out)
	}
	if got := inv.get(); !reflect.DeepEqual(got, [][2]string{{"set", c.Handle}}) {
		t.Errorf("invoked %v", got)
	}
}

func TestSetNeedsAnArgAndTheOthersTakeNone(t *testing.T) {
	r, _, ord, _ := spRig(t, nil)
	r.Srv.Err("n4_invoke_action", Args{"ord": ord, "action": "set"})
	r.Srv.Err("n4_invoke_action", Args{"ord": ord, "action": "set", "arg": 1.0, "arg_type": "baja:Int"})
	r.Srv.Err("n4_invoke_action", Args{"ord": ord, "action": "auto", "arg": 1.0, "arg_type": "baja:Double"})
}

func TestActiveHasNoInverseAndThePlanSaysSo(t *testing.T) {
	r, _, ord, _ := spRig(t, nil)
	plan := r.Srv.Dry("n4_invoke_action", Args{"ord": ord, "action": "active"})
	inverse, isList := Dig(plan, "plan", "inverse").([]any)
	said := false
	for _, n := range Dig(plan, "plan", "notes").([]any) {
		if s, _ := n.(string); strings.Contains(s, "no inverse") {
			said = true
		}
	}
	if !isList || len(inverse) != 0 || !said {
		t.Errorf("plan = %v", plan["plan"])
	}
}

func TestStateChangingActionsAreUnverifiedNotVerified(t *testing.T) {
	r, c, ord, inv := spRig(t, nil)
	for _, action := range []string{"active", "inactive", "auto"} {
		if out := r.Srv.RunWrite("n4_invoke_action", Args{"ord": ord, "action": action}); out["verdict"] != "unverified" {
			t.Errorf("%s: out = %v", action, out)
		}
	}
	want := [][2]string{{"active", c.Handle}, {"inactive", c.Handle}, {"auto", c.Handle}}
	if got := inv.get(); !reflect.DeepEqual(got, want) {
		t.Errorf("invoked %v", got)
	}
}

func TestSetReadBackMismatchIsReported(t *testing.T) {
	lying := false
	r, _, ord, _ := spRig(t, func(_ *boxstation.Station, action string, n *boxstation.Node) {
		if lying && action == "set" {
			n.Child("fallback").Child("value").Value = "1.0"
		}
	})
	args := withArgs(setSeven, Args{"ord": ord})
	plan := r.Srv.Dry("n4_invoke_action", args)
	r.Station.Do(func(*boxstation.Station) { lying = true })
	out := r.Srv.OK("n4_invoke_action", Execute(args, plan["confirmation_token"]))
	if out["verdict"] != "mismatch" {
		t.Errorf("out = %v", out)
	}
}
