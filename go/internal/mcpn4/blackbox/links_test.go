package blackbox

// n4_list_links (port of test_server.TestListLinks).

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// linkedPair adds A and B with an out slot and links A.out to B.in10; it returns the link name.
func linkedPair(r *Rig) (a, b Component, link string) {
	a, b = r.AddWithOut("A", 1), r.AddWithOut("B", 1)
	return a, b, r.Link(a, "out", b, "in10")
}

func links(r *Rig, args Args) []map[string]any {
	return objects(r.Srv.OK("n4_list_links", args)["links"])
}

func TestListsLinkWithResolvedSourcePath(t *testing.T) {
	r := stationRig(t)
	a, b, _ := linkedPair(r)
	want := []map[string]any{{"target_path": "/Folder/" + b.Name, "link_name": "Link",
		"source_ord": "h:" + a.Handle, "source_path": "/Folder/" + a.Name, "source_slot": "out",
		"target_slot": "in10"}}
	if got := links(r, Args{"ord": Folder}); !reflect.DeepEqual(got, want) {
		t.Errorf("links = %v", got)
	}
}

func TestSourceOutsideTheLoadedSubtreeHasNullPath(t *testing.T) {
	r := stationRig(t)
	a, b, _ := linkedPair(r)
	got := links(r, Args{"ord": Folder + "/" + b.Name, "depth": 1})
	if len(got) != 1 || got[0]["source_path"] != nil || got[0]["source_ord"] != "h:"+a.Handle {
		t.Errorf("links = %v", got)
	}
}

func TestNoLinksGivesEmptyList(t *testing.T) {
	r := stationRig(t)
	r.AddWithOut("A", 1)
	if got := r.Srv.OK("n4_list_links", Args{"ord": Folder})["links"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("links = %v", got)
	}
}

func TestConversionLinksAreIncluded(t *testing.T) {
	r := stationRig(t)
	_, b, link := linkedPair(r)
	r.Station.Do(func(st *boxstation.Station) { st.ByHandle(b.Handle).Child(link).Type = "baja:ConversionLink" })
	if got := links(r, Args{"ord": Folder}); len(got) != 1 {
		t.Errorf("links = %v", got)
	}
}

func TestDepthLimitsHowDeepComponentsAreScanned(t *testing.T) {
	r := stationRig(t)
	linkedPair(r)
	// from the root, B sits two levels down: depth 1 does not scan it, depth 2 does
	if got := links(r, Args{"ord": "station:|slot:/", "depth": 1}); len(got) != 0 {
		t.Errorf("depth 1: %v", got)
	}
	if got := links(r, Args{"ord": "station:|slot:/", "depth": 2}); len(got) != 1 {
		t.Errorf("depth 2: %v", got)
	}
}
