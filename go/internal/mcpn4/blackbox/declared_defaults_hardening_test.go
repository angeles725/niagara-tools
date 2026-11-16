package blackbox

// Declared defaults degrade to notes (review of #678): one default that cannot be decoded is left
// out with a note naming it, and the children of a declared struct default are marked default.

import (
	"strings"
	"testing"
)

func TestAnUndecodableDefaultIsLeftOutWithANoteNamingIt(t *testing.T) {
	r := stationRig(t)
	r.Station.SetRegistry(map[string]any{
		"kitBench:Odd": map[string]any{"p": "baja:Component", "c": []any{
			dfxProp("bad", map[string]any{"nm": "p", "t": "baja:StatusNumeric", "d": "?", "s": []any{
				map[string]any{"nm": "p", "n": "status", "v": "zz"}}}),
			dfxProp("ok", map[string]any{"nm": "p", "t": "baja:Double", "v": "0.0"})}},
		"baja:Component": map[string]any{}})
	c := r.Add("3", "Odd", "kitBench:Odd")
	for _, tool := range []string{"n4_read_slots", "n4_navigate"} {
		out := r.Srv.OK(tool, Args{"ord": Folder + "/" + c.Name})
		list := out["slots"]
		if tool == "n4_navigate" {
			list = out["children"]
		}
		if entries := objects(list); len(entries) != 1 || entries[0]["name"] != "ok" {
			t.Errorf("%s entries = %v, want ok alone", tool, list)
		}
		if note, _ := out["defaults_note"].(string); !strings.Contains(note, "bad") {
			t.Errorf("%s defaults_note = %v, want it to name bad", tool, out["defaults_note"])
		}
	}
}

func TestTheChildrenOfADeclaredStructDefaultAreMarkedDefault(t *testing.T) {
	r, c := dfxRig(t, stationRig)
	out := r.Srv.OK("n4_navigate", Args{"ord": Folder, "depth": 3})
	for _, dash := range objects(out["children"]) {
		if dash["name"] != c.Name {
			continue
		}
		for _, e := range objects(dash["children"]) {
			if e["name"] != "in1" {
				continue
			}
			kids := objects(e["children"])
			if len(kids) != 2 {
				t.Fatalf("in1 children = %v", e["children"])
			}
			for _, k := range kids {
				if k["default"] != true || k["handle"] != nil {
					t.Errorf("in1/%s = %v, want default true and handle null", k["name"], k)
				}
			}
			return
		}
	}
	t.Fatalf("no in1 under %v", out["children"])
}
