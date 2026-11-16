package blackbox

// Declared properties at their default (lab retro 2026-11-15 rows 1-2): a load omits a frozen
// property still at its type default, so n4_read_slots and n4_navigate list it from the type
// contract (the registry's loadTypes), marked default, and never invent a value.

import (
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

// dfxProp is a declared property of a contract with its default value node.
func dfxProp(name string, v map[string]any) map[string]any {
	p := map[string]any{"st": "p", "n": name}
	if v != nil {
		p["v"] = v
	}
	return p
}

// dfxContracts declare kitBench:Dash (setpoint, flag, an action) below kitBench:Base (in1, a
// Status default; proxyExt, a component default the station does not send; label, with no typed
// default), below baja:Component, which declares nothing ({}).
func dfxContracts() map[string]any {
	in1 := map[string]any{"nm": "p", "t": "baja:StatusNumeric", "d": "- {null}", "s": []any{
		map[string]any{"nm": "p", "n": "value", "v": "0.0", "d": "0,00"},
		map[string]any{"nm": "p", "n": "status", "v": "40", "d": "{null}"}}}
	return map[string]any{
		"kitBench:Dash": map[string]any{"p": "kitBench:Base", "c": []any{
			dfxProp("setpoint", map[string]any{"nm": "p", "t": "baja:Double", "d": "0,00", "v": "0.0"}),
			dfxProp("flag", map[string]any{"nm": "p", "t": "baja:Boolean", "d": "false", "v": "false"}),
			map[string]any{"st": "a", "n": "reset"}}},
		"kitBench:Base": map[string]any{"p": "baja:Component", "c": []any{
			dfxProp("in1", in1),
			dfxProp("proxyExt", map[string]any{"nm": "p", "t": "control:NullProxyExt", "d": "null", "nc": "true"}),
			dfxProp("label", nil)}},
		"baja:Component": map[string]any{},
	}
}

// dfxRig is a station whose Folder holds a fresh kitBench:Dash, the registry describing it.
func dfxRig(t *testing.T, rig func(*testing.T) *Rig) (*Rig, Component) {
	r := rig(t)
	r.Station.SetRegistry(dfxContracts())
	return r, r.Add("3", "Dash", "kitBench:Dash")
}

func TestReadSlotsListsTheDeclaredPropertiesTheLoadOmits(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	var names []string
	byName := map[string]map[string]any{}
	for _, s := range objects(out["slots"]) {
		names = append(names, s["name"].(string))
		byName[s["name"].(string)] = s
	}
	// inherited first, as declared; the action is not a property
	wantStrings(t, "slots", names, []string{"in1", "proxyExt", "label", "setpoint", "flag"})
	want := map[string]map[string]any{
		"setpoint": {"name": "setpoint", "type": "baja:Double", "value": "0.0", "value_display": "0,00", "status": nil, "default": true},
		"flag":     {"name": "flag", "type": "baja:Boolean", "value": "false", "value_display": "false", "status": nil, "default": true},
		"in1": {"name": "in1", "type": "baja:StatusNumeric", "value": "0.0", "value_display": "- {null}", "status": "40",
			"status_ok": false, "status_null": true, "default": true},
	}
	for name, w := range want {
		if !reflect.DeepEqual(byName[name], w) {
			t.Errorf("%s = %v\nwant %v", name, byName[name], w)
		}
	}
	for _, name := range []string{"proxyExt", "label"} {
		s := byName[name]
		if s["value"] != nil || s["default"] != true || !strings.Contains(s["note"].(string), "no default value") {
			t.Errorf("%s = %v, want value null with a note", name, s)
		}
	}
	if _, has := out["defaults_note"]; has {
		t.Errorf("defaults_note with readable contracts: %v", out["defaults_note"])
	}
}

func TestADeclaredPropertyTheStationSendsIsListedOnceWithItsValue(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	r.Set(c.Handle, "setpoint", map[string]any{"nm": "p", "t": "baja:Double", "v": "21.5"})
	n := 0
	for _, s := range objects(r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})["slots"]) {
		if s["name"] == "setpoint" {
			n++
			if s["value"] != "21.5" || s["default"] != nil {
				t.Errorf("setpoint = %v", s)
			}
		}
	}
	if n != 1 {
		t.Errorf("setpoint listed %d times", n)
	}
}

func TestTheContractsAreReadOncePerSession(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	args := Args{"ord": Folder + "/" + c.Name}
	r.Srv.OK("n4_read_slots", args)
	count := func() int {
		before := r.Station.Requests()
		r.Srv.OK("n4_read_slots", args)
		return r.Station.Requests() - before
	}
	again, plain := count(), 0
	r.Station.SetRegistry(map[string]any{})
	other := r.Add("3", "Plain", "kitBench:Plain") // answered {}: one registry call, then none
	r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + other.Name})
	before := r.Station.Requests()
	r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + other.Name})
	plain = r.Station.Requests() - before
	if again != plain {
		t.Errorf("a repeated read costs %d requests, a read of a type with no contract %d", again, plain)
	}
}

func TestATypeTheStationCannotDescribeSaysSoAndListsWhatWasLoaded(t *testing.T) {
	r := stationRig(t)
	r.Station.SetRegistry(map[string]any{"kitBench:Gone": map[string]any{"isErr": true, "c": "", "m": "kitBench:Gone", "s": ""}})
	c := r.Add("3", "Gone", "kitBench:Gone")
	r.Set(c.Handle, "label", map[string]any{"nm": "p", "t": "baja:String", "v": "x"})
	out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
	if slots := objects(out["slots"]); len(slots) != 1 || slots[0]["name"] != "label" {
		t.Errorf("slots = %v", out["slots"])
	}
	if note, _ := out["defaults_note"].(string); !strings.Contains(note, "kitBench:Gone") {
		t.Errorf("defaults_note = %v", out["defaults_note"])
	}
}

// A type the station does not describe (an error entry, or left out of the reply) is remembered
// for the session too: three reads ask the registry once.
func TestAnUndescribedTypeIsAskedForOncePerSession(t *testing.T) {
	for _, entry := range []any{nil, map[string]any{"isErr": true, "c": "", "m": "kitBench:Gone", "s": ""}} {
		r := stationRig(t)
		r.Station.SetRegistry(map[string]any{"kitBench:Gone": entry})
		c := r.Add("3", "Gone", "kitBench:Gone")
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
			out := r.Srv.OK("n4_read_slots", Args{"ord": Folder + "/" + c.Name})
			if note, _ := out["defaults_note"].(string); !strings.Contains(note, "kitBench:Gone") {
				t.Errorf("read %d defaults_note = %v", i, out["defaults_note"])
			}
		}
		if n := atomic.LoadInt32(&calls); n != 1 {
			t.Errorf("entry %v: three reads asked the registry %d times, want 1", entry, n)
		}
	}
}
