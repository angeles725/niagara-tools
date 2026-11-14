package blackbox

// n4_navigate (port of test_server.TestNavigate).

import (
	"reflect"
	"strings"
	"testing"
)

func TestDefaultListsRootChildren(t *testing.T) {
	want := map[string]any{"name": "Folder", "type": "baja:Folder", "handle": "3", "has_children": false}
	if got := named(stationRig(t).Srv.OK("n4_navigate", nil)["children"], "Folder"); !reflect.DeepEqual(got, want) {
		t.Errorf("Folder = %v", got)
	}
}

func TestHasChildrenTrueWhenChildHoldsComponents(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "Pump", "")
	if got := named(r.Srv.OK("n4_navigate", nil)["children"], "Folder"); got["has_children"] != true {
		t.Errorf("Folder = %v", got)
	}
}

func TestStatusSlotInternalsAreNotComponentChildren(t *testing.T) {
	r := stationRig(t)
	c := r.AddWithOut("Calc", 1)
	calc := named(r.Srv.OK("n4_navigate", Args{"ord": Folder})["children"], c.Name)
	if calc == nil || calc["has_children"] != false {
		t.Errorf("Calc = %v", calc)
	}
}

func TestNavigateIntoAFolder(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Pump", "")
	out := r.Srv.OK("n4_navigate", Args{"ord": Folder})
	got := [][]any{field(out["children"], "name"), field(out["children"], "type"), field(out["children"], "handle")}
	want := [][]any{{c.Name}, {"kitControl:NumericConst"}, {c.Handle}}
	if out["ord"] != Folder || !reflect.DeepEqual(got, want) {
		t.Errorf("navigate = %v", out)
	}
}

func TestDepthTwoNestsGrandchildren(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Pump", "")
	folder := named(r.Srv.OK("n4_navigate", Args{"depth": 2})["children"], "Folder")
	if got := field(folder["children"], "name"); !reflect.DeepEqual(got, []any{c.Name}) {
		t.Errorf("Folder children %v", got)
	}
}

func TestDepthOneHasNoNestedChildrenKey(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "Pump", "")
	children := objects(r.Srv.OK("n4_navigate", nil)["children"])
	if len(children) == 0 {
		t.Fatal("no children: the check below would be vacuous")
	}
	for _, child := range children {
		if _, has := child["children"]; has {
			t.Errorf("%v has children", child["name"])
		}
	}
}

func TestTypesFilterKeepsOnlyMatchingChildren(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Pump", "")
	r.Add("3", "Sub", "baja:Folder")
	out := r.Srv.OK("n4_navigate", Args{"ord": Folder, "types": []any{"kitControl:NumericConst"}})
	got := [][]any{field(out["children"], "name"), field(out["children"], "type"), field(out["children"], "matched")}
	want := [][]any{{c.Name}, {"kitControl:NumericConst"}, {true}}
	if !reflect.DeepEqual(got, want) || !reflect.DeepEqual(out["types"], []any{"kitControl:NumericConst"}) {
		t.Errorf("navigate = %v", out)
	}
}

func TestTypesFilterKeepsThePathToADeeperMatch(t *testing.T) {
	r := stationRig(t)
	c := r.Add("3", "Pump", "")
	out := r.Srv.OK("n4_navigate", Args{"depth": 2, "types": []any{"kitControl:NumericConst"}})
	folder := named(out["children"], "Folder")
	if !reflect.DeepEqual(field(out["children"], "name"), []any{"Folder"}) || folder["matched"] != false ||
		!reflect.DeepEqual(field(folder["children"], "name"), []any{c.Name}) ||
		!reflect.DeepEqual(field(folder["children"], "matched"), []any{true}) {
		t.Errorf("navigate = %v", out)
	}
}

func TestTypesFilterWithNoMatchReturnsNoChildren(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "Pump", "")
	out := r.Srv.OK("n4_navigate", Args{"depth": 2, "types": []any{"bacnet:BacnetDevice"}})
	if !reflect.DeepEqual(out["children"], []any{}) {
		t.Errorf("children %v", out["children"])
	}
}

func TestNoTypesFilterIsTheUnfilteredListing(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "Pump", "")
	for _, args := range []Args{{"depth": 2}, {"depth": 2, "types": []any{}}} {
		out := r.Srv.OK("n4_navigate", args)
		if _, has := out["types"]; has || named(out["children"], "Folder") == nil {
			t.Errorf("%v: %v", args, out)
		}
		for _, child := range objects(out["children"]) {
			if _, has := child["matched"]; has {
				t.Errorf("%v: %v has matched", args, child["name"])
			}
		}
	}
}

func TestTypesFilterRejectsNonStringEntries(t *testing.T) {
	r := stationRig(t)
	for _, types := range [][]any{{"baja:Folder", 5}, {""}} {
		if text := r.Srv.Err("n4_navigate", Args{"types": types}); !strings.Contains(text, "types") {
			t.Errorf("%v: %q", types, text)
		}
	}
}

func TestTypesFilterMustBeAnArray(t *testing.T) {
	if code := stationRig(t).Srv.CallError("n4_navigate", Args{"types": "baja:Folder"}); code != -32602 {
		t.Errorf("code %d", code)
	}
}

func TestBadOrdIsAnError(t *testing.T) {
	if text := stationRig(t).Srv.Err("n4_navigate", Args{"ord": "/Folder"}); !strings.Contains(text, "ord") {
		t.Errorf("text %q", text)
	}
}

func TestUnknownNodeIsAnError(t *testing.T) {
	stationRig(t).Srv.Err("n4_navigate", Args{"ord": "station:|slot:/Missing"})
}
