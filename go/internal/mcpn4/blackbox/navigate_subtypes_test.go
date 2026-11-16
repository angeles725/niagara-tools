package blackbox

// The n4_navigate types filter matches a type and its subtypes (second bench retro, delta 13):
// the ancestry comes from the station's type registry, one call for the types of the listing.

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// rfxHistoryRig is a station whose Folder holds a history extension (two levels below
// history:HistoryExt), a BACnet device and a constant; the registry knows their supertypes.
func rfxHistoryRig(t *testing.T) (r *Rig, ext, device, pump Component) {
	r = stationRig(t)
	for _, pair := range [][2]string{{"history:NumericIntervalHistoryExt", "history:IntervalHistoryExt"},
		{"history:IntervalHistoryExt", "history:HistoryExt"}, {"history:HistoryExt", "baja:Component"},
		{"bacnet:BacnetDevice", "driver:Device"}, {"kitControl:NumericConst", "kitControl:Constant"}} {
		r.Station.SetSupertype(pair[0], pair[1])
	}
	ext = r.Add("3", "NumericInterval", "history:NumericIntervalHistoryExt")
	device = r.Add("3", "Ahu", "bacnet:BacnetDevice")
	return r, ext, device, r.Add("3", "Pump", "")
}

// rfxMatches are the [name, type, matched] of each child of the Folder kept by types.
func rfxMatches(r *Rig, types ...any) (out [][]any, reply map[string]any) {
	reply = r.Srv.OK("n4_navigate", Args{"ord": Folder, "types": types})
	for _, c := range objects(reply["children"]) {
		out = append(out, []any{c["name"], c["type"], c["matched"]})
	}
	return out, reply
}

func TestTypesFilterMatchesASubtypeOfAGivenType(t *testing.T) {
	r, ext, device, _ := rfxHistoryRig(t)
	got, reply := rfxMatches(r, "history:HistoryExt")
	if want := [][]any{{ext.Name, "history:NumericIntervalHistoryExt", true}}; !reflect.DeepEqual(got, want) {
		t.Errorf("history:HistoryExt kept %v, want %v", got, want)
	}
	if _, has := reply["types_note"]; has {
		t.Errorf("types_note with a readable registry: %v", reply["types_note"])
	}
	got, _ = rfxMatches(r, "driver:Device", "history:IntervalHistoryExt")
	want := [][]any{{ext.Name, "history:NumericIntervalHistoryExt", true}, {device.Name, "bacnet:BacnetDevice", true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("two supertypes kept %v, want %v", got, want)
	}
}

func TestTypesFilterDoesNotMatchASupertypeOrASibling(t *testing.T) {
	r, _, _, _ := rfxHistoryRig(t)
	r.Add("3", "Base", "history:HistoryExt")
	if got, _ := rfxMatches(r, "history:NumericIntervalHistoryExt", "history:BooleanCovHistoryExt"); len(got) != 1 ||
		got[0][1] != "history:NumericIntervalHistoryExt" {
		t.Errorf("kept %v", got)
	}
}

func TestTypesFilterWithTheExactTypeAsksTheRegistryNothing(t *testing.T) {
	r, _, _, pump := rfxHistoryRig(t)
	r.Station.Do(func(st *boxstation.Station) { // only Pump is left: every listed type is an exact match
		folder := st.ByHandle("3")
		folder.Children = folder.Children[len(folder.Children)-1:]
	})
	rfxMatches(r, "kitControl:NumericConst") // the session is open
	before := r.Station.Requests()
	got, reply := rfxMatches(r, "kitControl:NumericConst")
	want := [][]any{{pump.Name, "kitControl:NumericConst", true}}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(reply["types"], []any{"kitControl:NumericConst"}) {
		t.Errorf("kept %v, types %v", got, reply["types"])
	}
	first := r.Station.Requests() - before
	r.Station.SetSupertype("kitControl:NumericConst", "")
	rfxMatches(r, "kitControl:Constant")
	if with := r.Station.Requests() - before - first; with != first+1 {
		t.Errorf("a navigate costs %d requests with exact types and %d with one type to look up, want one more", first, with)
	}
}

func TestTypesFilterFallsBackToExactTypesWhenTheRegistryCannotBeRead(t *testing.T) {
	r, ext, _, _ := rfxHistoryRig(t)
	r.Station.SetHook(func(frame map[string]any) (int, []byte) {
		msgs, _ := frame["m"].([]any)
		for _, m := range objects(msgs) {
			if m["c"] == "reg" {
				return 200, []byte(`{"v": "2.3", "p": "box", "m": [{"c": "reg", "k": "loadTypes", "r": 0, "t": "e", "b": {"isErr": true, "m": "no registry"}}]}`)
			}
		}
		return 0, nil
	})
	got, reply := rfxMatches(r, "history:HistoryExt", "history:NumericIntervalHistoryExt")
	want := [][]any{{ext.Name, "history:NumericIntervalHistoryExt", true}}
	note, _ := reply["types_note"].(string)
	if !reflect.DeepEqual(got, want) || note == "" {
		t.Errorf("kept %v, types_note %q", got, note)
	}
	if got, reply := rfxMatches(r, "history:HistoryExt"); len(got) != 0 || reply["types_note"] != note {
		t.Errorf("a supertype alone kept %v, types_note %v", got, reply["types_note"])
	}
}
