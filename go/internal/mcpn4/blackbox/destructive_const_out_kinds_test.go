package blackbox

// Regression probes for the constant-out delta: every constant sibling kind is carried by copy
// and by the remove snapshot, a changed value reads mismatch, computed or undeclared outs stay
// left out, and a station that does not describe the type keeps the baseline plan.

import (
	"encoding/json"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// kindContracts declares out summary-only on the four constants, and r / t on two probe types.
var kindContracts = map[string]any{
	"kitControl:NumericConst": outContract("s", "baja:StatusNumeric"),
	"kitControl:BooleanConst": outContract("s", "baja:StatusBoolean"),
	"kitControl:EnumConst":    outContract("s", "baja:StatusEnum"),
	"kitControl:StringConst":  outContract("s", "baja:StatusString"),
	"acme:ReadonlyOut":        outContract("r", "baja:StatusNumeric"),
	"acme:TransientOut":       outContract("t", "baja:StatusNumeric"),
}

func outContract(flags, t string) map[string]any {
	return map[string]any{"c": []any{map[string]any{"st": "p", "n": "out", "f": flags,
		"v": map[string]any{"nm": "p", "t": t}}}, "p": "baja:Component"}
}

// withKindContracts answers the registry channel from kindContracts ({} otherwise), leaving omit out.
func (d *dRig) withKindContracts(omit string) {
	d.f.set(func(f *faults) {
		f.raw = func(frame map[string]any) (int, []byte) {
			for _, m := range objects(frame["m"]) {
				if m["c"] != "reg" {
					continue
				}
				body, _ := m["b"].(map[string]any)
				out := map[string]any{}
				for _, s := range body["t"].([]any) {
					if spec, _ := s.(string); spec != omit {
						if c, ok := kindContracts[spec]; ok {
							out[spec] = c
						} else {
							out[spec] = map[string]any{}
						}
					}
				}
				data, _ := json.Marshal(map[string]any{"v": "2.3", "p": "box", "n": frame["n"], "m": []any{
					map[string]any{"c": "reg", "k": "loadTypes", "r": 0, "t": "rp", "b": out}}})
				return 200, data
			}
			return 0, nil
		}
	})
}

func statusOut(typ, vt, v, status string) map[string]any {
	return map[string]any{"nm": "p", "n": "out", "t": typ, "s": []any{
		map[string]any{"nm": "p", "n": "value", "t": vt, "v": v},
		map[string]any{"nm": "p", "n": "status", "t": "baja:Status", "v": status}}}
}

var constKinds = []struct{ typ, st, vt, v, status string }{
	{"kitControl:NumericConst", "baja:StatusNumeric", "baja:Double", "10.0", "0"},
	{"kitControl:BooleanConst", "baja:StatusBoolean", "baja:Boolean", "true", "0"},
	{"kitControl:EnumConst", "baja:StatusEnum", "baja:DynamicEnum", "2@{a=1,b=2}", "0"},
	{"kitControl:StringConst", "baja:StatusString", "baja:String", "hello", "0"},
	{"kitControl:NumericConst", "baja:StatusNumeric", "baja:Double", "7.5", "8"},
}

func (d *dRig) addConst(parent, name, typ string, out map[string]any) string {
	d.T.Helper()
	return d.addBody(parent, name, map[string]any{"nm": "p", "t": typ, "s": []any{out}})
}

// S1
func TestACopyCarriesTheOutOfEveryConstantKind(t *testing.T) {
	for _, k := range constKinds {
		t.Run(k.typ+"="+k.v, func(t *testing.T) {
			d := destructiveRig(t, nil)
			d.withKindContracts("")
			nn := d.addConst("3", "K", k.typ, statusOut(k.st, k.vt, k.v, k.status))
			done := d.Srv.RunWrite("n4_copy_component", copyConstArgs(nn, "K2"))
			got := d.childValues("K2", "out")
			if done["verdict"] != "verified" || got["value"] != k.v || (k.status != "0" && got["status"] != k.status) {
				t.Errorf("verdict %v, K2.out %v, want value %s status %s", done["verdict"], got, k.v, k.status)
			}
		})
	}
}

// S1
func TestACopyWhoseOutValueChangedIsAMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withKindContracts("")
	src := d.AddWithOut("K", 10.0)
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) {
		if o := n.Child("out"); o != nil && o.Child("value") != nil {
			o.Child("value").Value = "3.0"
		}
	})
	if done := d.Srv.RunWrite("n4_copy_component", copyConstArgs(src.Name, "K2")); done["verdict"] != "mismatch" {
		t.Errorf("verdict = %v", done["verdict"])
	}
}

// S2
func TestRollbackOfARemoveRestoresTheOutOfEveryConstantKind(t *testing.T) {
	for _, k := range constKinds {
		t.Run(k.typ+"="+k.v, func(t *testing.T) {
			d := destructiveRig(t, nil)
			d.withKindContracts("")
			nn := d.addConst("3", "K", k.typ, statusOut(k.st, k.vt, k.v, k.status))
			removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
			back := d.rollback(removed["batch_id"])
			got := d.childValues(nn, "out")
			if back["verdict"] != "verified" || got["value"] != k.v || (k.status != "0" && got["status"] != k.status) {
				t.Errorf("verdict %v, out %v, want value %s status %s", back["verdict"], got, k.v, k.status)
			}
		})
	}
}

// S2
func TestRollbackThatRestoresADifferentOutValueIsAMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withKindContracts("")
	k := d.AddWithOut("K", 10.0)
	removed := d.Srv.RunWrite("n4_remove_component", removeArgs(k.Name))
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) {
		if o := n.Child("out"); o != nil && o.Child("value") != nil {
			o.Child("value").Value = "0.5"
		}
	})
	if back := d.rollback(removed["batch_id"]); back["verdict"] != "mismatch" {
		t.Errorf("verdict = %v", back["verdict"])
	}
}

// S2
func TestRollbackOfARemovedFolderChecksItsNestedConstantOut(t *testing.T) {
	for _, drop := range []bool{false, true} {
		d := destructiveRig(t, nil)
		d.withKindContracts("")
		fn := d.addBody("3", "F", map[string]any{"nm": "p", "t": "baja:Folder"})
		d.addConst(d.node(fn).Handle, "K", "kitControl:NumericConst",
			statusOut("baja:StatusNumeric", "baja:Double", "10.0", "0"))
		removed := d.Srv.RunWrite("n4_remove_component", removeArgs(fn))
		if drop {
			d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) { removeChild(n, "out") })
		}
		back := d.rollback(removed["batch_id"])
		want, wantV := "verified", any("10.0")
		if drop {
			want, wantV = "mismatch", nil
		}
		if back["verdict"] != want || d.childValues(fn, "K", "out")["value"] != wantV {
			t.Errorf("drop=%v: verdict %v, K.out %v", drop, back["verdict"], d.childValues(fn, "K", "out"))
		}
	}
}

// S3
func TestAReadonlyTransientOrUndeclaredOutIsStillLeftOut(t *testing.T) {
	for _, typ := range []string{"acme:ReadonlyOut", "acme:TransientOut", "acme:Undeclared"} {
		t.Run(typ, func(t *testing.T) {
			d := destructiveRig(t, nil)
			d.withKindContracts("")
			nn := d.addConst("3", "W", typ, statusOut("baja:StatusNumeric", "baja:Double", "10.0", "0"))
			dry := d.Srv.Dry("n4_copy_component", copyConstArgs(nn, "W2"))
			if ops := objects(planOf(dry)["ops"]); len(ops) != 1 || outOf(ops[0]) != nil {
				t.Errorf("copy ops = %v", ops)
			}
			sent := d.f.record()
			removed := d.Srv.RunWrite("n4_remove_component", removeArgs(nn))
			d.rollback(removed["batch_id"])
			if op := addOf(sent(), nn); op == nil || destructiveContainsAny(field(Dig(op, "b", "s"), "n"), "out") {
				t.Errorf("re-create add = %v", op)
			}
		})
	}
}

// S4
func TestARemoveOfATypeTheStationDoesNotDescribeKeepsTheBaselineSnapshot(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withKindContracts("kitControl:NumericConst")
	k := d.AddWithOut("K", 10.0)
	dry := d.Srv.Dry("n4_remove_component", removeArgs(k.Name))
	inv := objects(planOf(dry)["inverse"])
	if len(inv) != 1 || Dig(inv[0], "b", "s") != nil {
		t.Errorf("inverse = %v, want the type alone", inv)
	}
}
