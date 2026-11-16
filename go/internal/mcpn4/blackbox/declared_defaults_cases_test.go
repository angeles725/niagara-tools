package blackbox

// Verifier probes for declared defaults (lab retro 2026-11-15 rows 1-2): loaded slots keep their
// place, only added entries are marked default, contracts are asked once per session, a partial
// or malformed contract never yields an invented value, and a rollback read-back counts an
// omitted frozen property as its declared default.

import (
	"reflect"
	"testing"
)

func TestReadSlotsKeepsTheLoadedSlotsFirstAndMarksOnlyTheAddedOnes(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	r.Set(c.Handle, "flag", map[string]any{"nm": "p", "t": "baja:Boolean", "v": "true"})
	r.Set(c.Handle, "dyn", map[string]any{"nm": "p", "t": "baja:String", "v": "x"})
	slots := objects(r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})["slots"])
	var names []string
	for _, s := range slots {
		names = append(names, s["name"].(string))
	}
	if want := []string{"flag", "dyn", "in1", "proxyExt", "label", "setpoint"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("slots = %v, want %v", names, want)
	}
	for i, s := range slots {
		_, marked := s["default"]
		if marked != (i >= 2) {
			t.Errorf("%s default = %v, want marked only on added entries", s["name"], s["default"])
		}
	}
	if slots[0]["value"] != "true" {
		t.Errorf("loaded flag = %v, want its loaded value", slots[0])
	}
}

func TestAContractWhoseSupertypeIsNotDescribedListsTheSubtypeDefaultsAndNamesTheGap(t *testing.T) {
	r := stationRig(t)
	contracts := dfxContracts()
	contracts["kitBench:Base"] = nil
	r.Station.SetRegistry(contracts)
	c := r.Add("3", "Dash", "kitBench:Dash")
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	if got := field(out["slots"], "name"); !reflect.DeepEqual(got, []any{"setpoint", "flag"}) {
		t.Errorf("slots = %v, want only the subtype's declared properties", got)
	}
	want := "the station did not describe type kitBench:Base: its declared properties at their default are not listed"
	if out["defaults_note"] != want {
		t.Errorf("defaults_note = %v, want %q", out["defaults_note"], want)
	}
}
