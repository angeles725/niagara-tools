package blackbox

// Verifier probes for declared defaults (lab retro 2026-11-15 rows 1-2): loaded slots keep their
// place, only added entries are marked default, contracts are asked once per session, a partial
// or malformed contract never yields an invented value, and a rollback read-back counts an
// omitted frozen property as its declared default.

import (
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const registryDown = `{"v": "2.3", "p": "box", "m": [{"c": "reg", "k": "loadTypes", "r": 0, "t": "e", "b": {"isErr": true, "m": "no registry"}}]}`

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

func TestRepliesOfAComponentWithNoOmittedPropertyCarryNoDefaults(t *testing.T) {
	r := stationRig(t)
	r.Station.SetRegistry(map[string]any{
		"kitBench:Full":  map[string]any{"p": "baja:Component", "c": []any{dfxProp("setpoint", map[string]any{"nm": "p", "t": "baja:Double", "v": "0.0"})}},
		"baja:Component": map[string]any{}})
	c := r.Add("3", "Full", "kitBench:Full")
	r.Set(c.Handle, "setpoint", map[string]any{"nm": "p", "t": "baja:Double", "v": "4.0"})
	read := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	want := []any{map[string]any{"name": "setpoint", "type": "baja:Double", "value": "4.0", "status": nil}}
	if !reflect.DeepEqual(read["slots"], want) {
		t.Errorf("slots = %v, want %v", read["slots"], want)
	}
	nav := r.Srv.OK("n4_navigate", Args{"ord": Folder + "/" + c.Name})
	wantNav := []any{map[string]any{"name": "setpoint", "type": "baja:Double", "handle": nil, "has_children": false}}
	if !reflect.DeepEqual(nav["children"], wantNav) {
		t.Errorf("children = %v, want %v", nav["children"], wantNav)
	}
	for _, out := range []map[string]any{read, nav} {
		if _, has := out["defaults_note"]; has {
			t.Errorf("defaults_note = %v", out["defaults_note"])
		}
	}
}

func TestKnownContractsAreNotAskedAgainAfterTheFirstReads(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	r.Srv.OK("n4_navigate", Args{"ord": Folder, "depth": 2})
	var calls int32
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		for _, m := range objects(frame["m"]) {
			if m["c"] == "reg" {
				atomic.AddInt32(&calls, 1)
			}
		}
		return 0, nil
	})
	for i := 0; i < 3; i++ {
		r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
		r.Srv.OK("n4_navigate", Args{"ord": Folder, "depth": 2})
	}
	if n := atomic.LoadInt32(&calls); n != 0 {
		t.Errorf("repeated reads asked the registry %d times, want 0", n)
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

func TestAMalformedContractDefaultIsListedWithoutAValue(t *testing.T) {
	r := stationRig(t)
	r.Station.SetRegistry(map[string]any{
		"kitBench:Bad": map[string]any{"p": "baja:Component", "c": []any{
			map[string]any{"st": "p", "n": "weird", "v": "garbage"},
			map[string]any{"st": "p", "n": "noType", "v": map[string]any{"nm": "p", "v": "1"}},
			map[string]any{"st": "p", "n": 7}}},
		"baja:Component": map[string]any{}})
	c := r.Add("3", "Bad", "kitBench:Bad")
	for _, tool := range []string{"n4_read_slots", "n4_navigate"} {
		out := r.Srv.OK(tool, Args{"ord": Folder + "/" + c.Name})
		list := out["slots"]
		if tool == "n4_navigate" {
			list = out["children"]
		}
		entries := objects(list)
		if len(entries) != 2 {
			t.Fatalf("%s entries = %v, want weird and noType", tool, list)
		}
		for _, e := range entries {
			if e["value"] != nil || e["default"] != true || e["note"] != noDefaultNoteText {
				t.Errorf("%s %v, want value null, default true and the note", tool, e)
			}
		}
	}
}

const noDefaultNoteText = "at its declared default, but the type contract gives no default value (a component default is not sent): read it in Workbench"

func TestNavigateWithATypesFilterAddsNoUntypedDefault(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	out := r.Srv.OK("n4_navigate", Args{"ord": Folder, "depth": 2, "types": []any{"baja:Double"}})
	dash := objects(out["children"])
	if len(dash) != 1 || dash[0]["name"] != c.Name || dash[0]["matched"] != false {
		t.Fatalf("children = %v, want only the path to the matching default", out["children"])
	}
	want := []any{map[string]any{"name": "setpoint", "type": "baja:Double", "handle": nil, "has_children": false,
		"default": true, "value": "0.0", "matched": true}}
	if !reflect.DeepEqual(dash[0]["children"], want) {
		t.Errorf("Dash children = %v, want %v", dash[0]["children"], want)
	}
}

// dropAfterRollbackWrite removes slot from Folder/Dash on every load that follows the rollback's
// syncTo, as a station omits a frozen property set back to its default; failReg answers every
// registry call with an error. The returned flag arms it.
func dropAfterRollbackWrite(r *Rig, slot string, failReg bool) *atomic.Bool {
	var armed, synced atomic.Bool
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		for _, m := range objects(frame["m"]) {
			if failReg && m["c"] == "reg" {
				return 200, []byte(registryDown)
			}
			b, _ := m["b"].(map[string]any)
			if !armed.Load() || b == nil {
				continue
			}
			if b["sck"] == "syncTo" {
				synced.Store(true)
			}
			if b["sck"] == "loadSlots" && synced.Load() {
				r.Station.Do(func(st *boxstation.Station) {
					dash := st.Root().Child("Folder").Child("Dash")
					for i, k := range dash.Children {
						if k.Name == slot {
							dash.Children = append(dash.Children[:i], dash.Children[i+1:]...)
							return
						}
					}
				})
			}
		}
		return 0, nil
	})
	return &armed
}

func TestARollbackToADefaultTheStationOmitsReadsVerified(t *testing.T) {
	r, c := dfxRig(t, verifiedRig)
	armed := dropAfterRollbackWrite(r, "flag", false)
	set := r.Srv.RunWrite("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "flag", "value": true, "value_type": "baja:Boolean"})
	armed.Store(true)
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": set["batch_id"]})
	if back["verdict"] != "verified" || Dig(back, "observed", "restored") != true {
		t.Errorf("rollback = %v, want verified and restored", back)
	}
}

func TestARollbackWhoseNonDefaultValueIsLostReadsMismatch(t *testing.T) {
	r, c := dfxRig(t, verifiedRig)
	r.Set(c.Handle, "setpoint", map[string]any{"nm": "p", "t": "baja:Double", "v": "3.0"})
	armed := dropAfterRollbackWrite(r, "setpoint", false)
	set := r.Srv.RunWrite("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "setpoint", "value": 5, "value_type": "baja:Double"})
	armed.Store(true)
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": set["batch_id"]})
	if back["verdict"] != "mismatch" || Dig(back, "observed", "restored") != false {
		t.Errorf("rollback = %v, want mismatch: the default 0.0 is not the recorded 3.0", back)
	}
}

func TestARollbackReadBackWithAnUnreadableRegistryReportsTheReadBackError(t *testing.T) {
	r, c := dfxRig(t, verifiedRig)
	set := r.Srv.RunWrite("n4_set_slot", Args{"ord": Folder + "/" + c.Name, "slot": "flag", "value": true, "value_type": "baja:Boolean"})
	r.Restart(nil) // a new session: no contract is cached
	r.ConnectVerified()
	armed := dropAfterRollbackWrite(r, "flag", true)
	armed.Store(true)
	back := r.Srv.RunWrite("n4_rollback", Args{"batch_id": set["batch_id"]})
	if back["verdict"] != "failed" || back["readback_failed"] != true || back["readback_error"] == nil {
		t.Errorf("rollback = %v, want the read-back error, not a mismatch, when no contract can be read", back)
	}
}
