package blackbox

// n4_bql_query on a projected column that resolves to no slot (retro 2026-11-19 M1): BQL answers
// "NULL" for a name it cannot resolve, so a facet key such as maxOverrideDuration read as 135 rows
// of "NULL". A column NULL on every row whose name is no slot of the queried type gets a warning.

import (
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const facetQuery = "select slotPath, name, maxOverrideDuration, out, fallback from control:NumericWritable"

// writableRig is a servlet rig whose registry describes control:NumericWritable (out, fallback)
// under control:ControlPoint (facets), answering facetQuery with csv.
func writableRig(t *testing.T, csv string) *Rig {
	r, sv := servletRig(t, nil)
	r.Station.SetRegistry(map[string]any{
		"control:NumericWritable": map[string]any{"p": "control:ControlPoint", "c": []any{
			map[string]any{"st": "p", "n": "out"}, map[string]any{"st": "p", "n": "fallback"}}},
		"control:ControlPoint": map[string]any{"c": []any{map[string]any{"st": "p", "n": "facets"}}},
	})
	routeOrds(sv, boxstation.OrdRoute{Needle: "NumericWritable", Body: []byte(csv)})
	return r
}

func TestAColumnNullOnEveryRowThatIsNoSlotIsWarned(t *testing.T) {
	r := writableRig(t, "Slot Path,Name,maxOverrideDuration,Out,Fallback\n"+
		"slot:/A,A,NULL,1.0,NULL\nslot:/B,B,NULL,NULL,NULL\n")
	out := r.Srv.OK("n4_bql_query", Args{"base": "/Folder", "query": facetQuery})
	warnings, _ := out["warnings"].([]any)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one (maxOverrideDuration only)", out["warnings"])
	}
	w, _ := warnings[0].(string)
	for _, part := range []string{"maxOverrideDuration", "no such slot", "control:NumericWritable", "facet?"} {
		if !strings.Contains(w, part) {
			t.Errorf("missing %q: %s", part, w)
		}
	}
	if out["row_count"] != float64(2) {
		t.Errorf("rows changed: %v", out)
	}
}

func TestAColumnNullOnSomeRowsOnlyIsNotWarned(t *testing.T) {
	r := writableRig(t, "Slot Path,Name,maxOverrideDuration,Out,Fallback\nslot:/A,A,5000,1.0,NULL\nslot:/B,B,NULL,NULL,2.0\n")
	if out := r.Srv.OK("n4_bql_query", Args{"base": "/Folder", "query": facetQuery}); out["warnings"] != nil {
		t.Errorf("warnings = %v", out["warnings"])
	}
}

func TestANormalBqlReplyIsUnchanged(t *testing.T) {
	r := writableRig(t, "Slot Path,Name,Out\nslot:/A,A,1.0\n")
	out := r.Srv.OK("n4_bql_query", Args{"base": "/Folder", "query": "select slotPath, name, out from control:NumericWritable"})
	keys := []string{}
	for k := range out {
		keys = append(keys, k)
	}
	want := map[string]bool{"base": true, "query": true, "columns": true, "row_count": true, "truncated": true,
		"elapsed_ms": true, "rows": true}
	got := map[string]bool{}
	for _, k := range keys {
		got[k] = true
	}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(out["rows"], []any{map[string]any{
		"Slot Path": "slot:/A", "Name": "A", "Out": "1.0"}}) {
		t.Errorf("reply = %v", out)
	}
}
