package blackbox

// A constant block's configured out (kitControl:NumericConst declares it with flags 8, summary
// only: neither readonly nor transient) is configuration: a copy carries it and its read-back
// compares it. A computed out (control:NumericWritable declares it rtso) is still left out.

import (
	"encoding/json"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// constContracts are the registry contracts the station answers for these tests; every other
// requested type is answered {} (a type with no declared slot), as the fake does.
var constContracts = map[string]any{
	"kitControl:NumericConst": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "out", "f": "s", "v": map[string]any{"nm": "p", "t": "baja:StatusNumeric"}}},
		"p": "baja:Component"},
	"control:NumericWritable": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "out", "f": "rtso", "v": map[string]any{"nm": "p", "t": "baja:StatusNumeric"}}},
		"p": "baja:Component"},
}

// withContracts makes the station's registry channel answer constContracts.
func (d *dRig) withContracts() {
	d.f.set(func(f *faults) {
		f.raw = func(frame map[string]any) (int, []byte) {
			for _, m := range objects(frame["m"]) {
				if m["c"] != "reg" {
					continue
				}
				body, _ := m["b"].(map[string]any)
				out := map[string]any{}
				for _, s := range body["t"].([]any) {
					spec, _ := s.(string)
					if c, ok := constContracts[spec]; ok {
						out[spec] = c
					} else {
						out[spec] = map[string]any{}
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

func copyConstArgs(name string, newNames ...any) Args {
	return Args{"parent_ord": Folder, "name": name, "new_names": newNames}
}

// outOf is the "out" child of a copy op's body, nil when the body has none.
func outOf(op any) map[string]any {
	return byName(Dig(op, "b", "s"))["out"]
}

func TestACopyOfANumericConstCarriesItsConfiguredOut(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	src := d.AddWithOut("K", 10.0)
	dry := d.Srv.Dry("n4_copy_component", copyConstArgs(src.Name, "K2"))
	ops := objects(planOf(dry)["ops"])
	if len(ops) != 1 || outOf(ops[0]) == nil {
		t.Fatalf("plan ops carry no out: %v", planOf(dry)["ops"])
	}
	done := d.Srv.OK("n4_copy_component", Execute(copyConstArgs(src.Name, "K2"), dry["confirmation_token"]))
	if done["verdict"] != "verified" {
		t.Errorf("verdict = %v (%v)", done["verdict"], done)
	}
	if got := d.childValues("K2", "out"); got["value"] != "10.0" {
		t.Errorf("K2.out = %v, want value 10.0", got)
	}
}

func TestACopyThatLosesTheCarriedOutIsAMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	src := d.AddWithOut("K", 10.0)
	d.afterAdd(func(_ *boxstation.Station, n *boxstation.Node) { removeChild(n, "out") }) // the station drops it
	done := d.Srv.RunWrite("n4_copy_component", copyConstArgs(src.Name, "K2"))
	if done["verdict"] != "mismatch" {
		t.Errorf("verdict = %v (%v)", done["verdict"], done)
	}
}

func TestACopyOfANumericWritableStillLeavesItsComputedOutOut(t *testing.T) {
	d := destructiveRig(t, nil)
	d.withContracts()
	src := d.Add("3", "W", "control:NumericWritable")
	d.Set(src.Handle, "out", StatusNumeric(10.0, "0"))
	dry := d.Srv.Dry("n4_copy_component", copyConstArgs(src.Name, "W2"))
	ops := objects(planOf(dry)["ops"])
	if len(ops) != 1 || outOf(ops[0]) != nil {
		t.Errorf("plan ops = %v, want one add without out", planOf(dry)["ops"])
	}
}
