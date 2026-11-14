package blackbox

// n4_find_dangling_outputs (port of test_server.TestDanglingOutputs and TestDeepTargets).

import (
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func dangling(r *Rig, args Args) map[string]any { return r.Srv.OK("n4_find_dangling_outputs", args) }

func TestReportsComponentsWhoseOutIsNotALinkSource(t *testing.T) {
	r := stationRig(t)
	_, b, _ := linkedPair(r)
	want := []any{map[string]any{"path": "/Folder/" + b.Name, "type": "kitControl:NumericConst"}}
	if got := dangling(r, Args{"ord": Folder})["dangling"]; !reflect.DeepEqual(got, want) {
		t.Errorf("dangling = %v", got)
	}
}

func TestScopeNoteWarnsAboutLinksFromOutside(t *testing.T) {
	out := dangling(stationRig(t), Args{"ord": Folder})
	if !reflect.DeepEqual(out["dangling"], []any{}) || !strings.Contains(jsonText(out["scope_note"]), "outside the subtree") {
		t.Errorf("out = %v", out)
	}
}

func TestComponentWithoutOutSlotIsNotReported(t *testing.T) {
	r := stationRig(t)
	r.Add("3", "NoOut", "")
	if got := dangling(r, Args{"ord": Folder})["dangling"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("dangling = %v", got)
	}
}

func TestLinkFromAnotherSlotDoesNotCoverOut(t *testing.T) {
	r := stationRig(t)
	a, b := r.AddWithOut("A", 1), r.AddWithOut("B", 1)
	r.Set(a.Handle, "alt", StatusNumeric(1, "0"))
	r.Link(a, "alt", b, "in10")
	want := []string{"/Folder/" + a.Name, "/Folder/" + b.Name}
	sort.Strings(want)
	if got := sortedStrings(dangling(r, Args{"ord": Folder})["dangling"], "path"); !reflect.DeepEqual(got, want) {
		t.Errorf("dangling paths %v", got)
	}
}

// linkWithSourceOrd is an A -> B link whose sourceOrd the station reports as sourceOrd.
func linkWithSourceOrd(r *Rig, sourceOrd string) (a, b Component) {
	a, b, link := linkedPair(r)
	r.Station.Do(func(st *boxstation.Station) { st.ByHandle(b.Handle).Child(link).Child("sourceOrd").Value = sourceOrd })
	return a, b
}

func TestASlotPathSourceIsResolvedLikeAHandle(t *testing.T) {
	r := stationRig(t)
	a, b := linkWithSourceOrd(r, "station:|slot:/Folder/A")
	out := dangling(r, Args{"ord": Folder})
	if !reflect.DeepEqual(field(out["dangling"], "path"), []any{"/Folder/" + b.Name}) || !reflect.DeepEqual(out["unknown"], []any{}) {
		t.Errorf("out = %v", out)
	}
	if got := links(r, Args{"ord": Folder}); len(got) == 0 || got[0]["source_path"] != "/Folder/"+a.Name {
		t.Errorf("links = %v", got)
	}
}

func TestAnUnresolvableSourceMakesCandidatesUnknownNotDangling(t *testing.T) {
	r := stationRig(t)
	const source = "local:|foxs:|station:|bql:select x from y"
	a, b := linkWithSourceOrd(r, source)
	out := dangling(r, Args{"ord": Folder})
	want := []string{"/Folder/" + a.Name, "/Folder/" + b.Name}
	sort.Strings(want)
	unresolved := []any{map[string]any{"target_path": "/Folder/" + b.Name, "link_name": "Link",
		"source_ord": source, "source_slot": "out"}}
	if !reflect.DeepEqual(out["dangling"], []any{}) || !reflect.DeepEqual(sortedStrings(out["unknown"], "path"), want) ||
		!reflect.DeepEqual(out["unresolved_links"], unresolved) || !strings.Contains(jsonText(out["scope_note"]), "unknown") {
		t.Errorf("out = %v", out)
	}
}

func TestAHandleSourceOutsideTheSubtreeKeepsCandidatesDangling(t *testing.T) {
	r := stationRig(t)
	_, b, _ := linkedPair(r)
	out := dangling(r, Args{"ord": Folder + "/" + b.Name, "depth": 1})
	want := []any{map[string]any{"path": "/Folder/" + b.Name, "type": "kitControl:NumericConst"}}
	if !reflect.DeepEqual(out["dangling"], want) || !reflect.DeepEqual(out["unknown"], []any{}) {
		t.Errorf("out = %v", out)
	}
}

// deepTargets is TestDeepTargets' station: A in the Folder links to B inside C, one level
// deeper than depth 1 scans.
func deepTargets(t *testing.T) (r *Rig, a, b, c Component) {
	r = stationRig(t)
	a, c = r.AddWithOut("A", 1), r.Add("3", "C", "")
	b = r.Add(c.Handle, "B", "")
	r.Link(a, "out", b, "in10")
	return r, a, b, c
}

func TestOutputFeedingADeeperTargetIsNotDangling(t *testing.T) {
	r, a, _, _ := deepTargets(t)
	out := dangling(r, Args{"ord": Folder, "depth": 1})
	if _, isList := out["dangling"].([]any); !isList {
		t.Fatalf("no dangling list: %v", out)
	}
	if paths := field(out["dangling"], "path"); slices.Contains(paths, any("/Folder/"+a.Name)) {
		t.Errorf("dangling %v", paths)
	}
}

func TestSourceInTheReportedRangeResolvesForADeeperTarget(t *testing.T) {
	r, a, b, c := deepTargets(t)
	got := links(r, Args{"ord": Folder, "depth": 2})
	if len(got) != 1 || got[0]["source_path"] != "/Folder/"+a.Name || got[0]["target_path"] != "/Folder/"+c.Name+"/"+b.Name {
		t.Errorf("links = %v", got)
	}
}

func TestListLinksReportsOnlyLinksOfComponentsWithinDepth(t *testing.T) {
	r, _, _, _ := deepTargets(t)
	if got := r.Srv.OK("n4_list_links", Args{"ord": Folder, "depth": 1})["links"]; !reflect.DeepEqual(got, []any{}) {
		t.Errorf("links = %v", got)
	}
}

func TestScopeNoteStatesWhatIsAndIsNotSeen(t *testing.T) {
	r, _, _, _ := deepTargets(t)
	note := jsonText(dangling(r, Args{"ord": Folder})["scope_note"])
	if !strings.Contains(note, "one level below depth") || !strings.Contains(note, "outside the subtree") {
		t.Errorf("scope_note %s", note)
	}
}
