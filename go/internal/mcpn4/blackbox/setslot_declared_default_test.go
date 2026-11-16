package blackbox

// n4_set_slot and the declared default (lab retro 2026-11-16 row M8): the station omits a frozen
// property whose value equals the default its type DECLARES (a 10 s RelTime, false), not only 0,
// false or an empty string. A write of that default is planned and read back from the type
// registry, so it reads verified; only a slot no contract declares is refused, and the refusal
// says which rule the station follows.

import (
	"strings"
	"testing"
)

// m8Contracts are dfxContracts with a RelTime property whose declared default is 10 s.
func m8Contracts() map[string]any {
	contracts := dfxContracts()
	dash := contracts["kitBench:Dash"].(map[string]any)
	dash["c"] = append(dash["c"].([]any),
		dfxProp("delay", map[string]any{"nm": "p", "t": "baja:RelTime", "d": "10 s", "v": "10000"}))
	return contracts
}

func TestAWriteOfTheDeclaredDefaultReadsBackVerified(t *testing.T) {
	for _, tc := range []struct {
		slot, valueType string
		value           any
		want            any
	}{
		{"delay", "baja:RelTime", 10000, "10000"},
		{"flag", "baja:Boolean", false, false},
	} {
		t.Run(tc.slot, func(t *testing.T) {
			r := verifiedRig(t)
			r.Station.SetRegistry(m8Contracts())
			c := r.Add("3", "Dash", "kitBench:Dash") // every declared property at its default: omitted
			armed := dropAfterRollbackWrite(r, tc.slot, false)
			armed.Store(true) // the station still omits the slot after the write: it holds the default
			set := r.Srv.RunWrite("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": tc.slot,
				"value": tc.value, "value_type": tc.valueType})
			if set["verdict"] != "verified" || Dig(set, "observed") != tc.want {
				t.Errorf("set = %v, want verified with observed %v", set, tc.want)
			}
		})
	}
}

func TestAnUndeclaredOmittedSlotIsRefusedWithTheDeclaredDefaultRule(t *testing.T) {
	r := verifiedRig(t)
	r.Station.SetRegistry(m8Contracts())
	c := r.Add("3", "Dash", "kitBench:Dash")
	text := r.Srv.Err("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "nope", "value": 1.0,
		"value_type": "baja:Double"})
	for _, needle := range []string{"not found", "equals its declared default", "10 s",
		"type registry", "no type contract", "n4_read_slots", "Workbench"} {
		if !strings.Contains(text, needle) {
			t.Errorf("refusal lacks %q: %s", needle, text)
		}
	}
	if strings.Contains(text, "type default (0.0, false, an empty string)") {
		t.Errorf("refusal keeps the old type-default claim: %s", text)
	}
}
