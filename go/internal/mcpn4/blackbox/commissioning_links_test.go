package blackbox

// n4_commissioning_report, the links and naming sections.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const comFolder = "slot:/Folder/"

func TestCommissioningListsLinksWhoseSourceDoesNotResolve(t *testing.T) {
	r, _ := comRig(t, nil)
	_, b, link := linkedPair(r)
	c := r.AddWithOut("C", 1)
	second := r.Link(b, "out", c, "in10")
	r.Station.Do(func(st *boxstation.Station) {
		st.ByHandle(b.Handle).Child(link).Child("sourceOrd").Value = "station:|slot:/Folder/Gone"
		st.ByHandle(c.Handle).Child(second).Child("sourceOrd").Value = "virtual:/x"
	})
	s := comSection(t, r, "links", Args{"scope": Folder})
	check := comWant(t, s, "unresolved_source", 2, comFolder+b.Name, comFolder+c.Name)
	if fact := comFact(check, comFolder+b.Name); !strings.Contains(fact, "/Folder/Gone does not exist") {
		t.Errorf("missing path fact = %q", fact)
	}
	if fact := comFact(check, comFolder+c.Name); !strings.Contains(fact, "virtual:/x") {
		t.Errorf("unresolved form fact = %q", fact)
	}
	// a source outside the scanned subtree is not judged
	r.Station.Do(func(st *boxstation.Station) { st.ByHandle(b.Handle).Child(link).Child("sourceOrd").Value = "h:ffff" })
	s = comSection(t, r, "links", Args{"scope": Folder})
	if c := comWant(t, s, "unresolved_source", 1, comFolder+c.Name); c["source_outside_scan"] != 1.0 {
		t.Errorf("outside = %v", c)
	}
}

func TestCommissioningListsAWritableInputFedByTwoLinks(t *testing.T) {
	r, _ := comRig(t, nil)
	a, b, _ := linkedPair(r)
	c := r.AddWithOut("C", 1)
	r.Link(c, "out", b, "in10")
	r.Link(a, "out", c, "routeAlarm") // an action fed by two links is not an input conflict
	r.Link(b, "out", c, "routeAlarm")
	s := comSection(t, r, "links", Args{"scope": Folder})
	check := comWant(t, s, "inputs_with_several_links", 1, comFolder+b.Name)
	if fact := comFact(check, comFolder+b.Name); fact != "slot in10 is the target of 2 links: /Folder/"+a.Name+".out, /Folder/"+c.Name+".out" {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningReportsTheDanglingOutputsOfTheExistingCheck(t *testing.T) {
	r, _ := comRig(t, nil)
	_, b, _ := linkedPair(r)
	s := comSection(t, r, "links", Args{"scope": Folder})
	comWant(t, s, "dangling_outputs", 1, comFolder+b.Name)
	want := r.Srv.OK("n4_find_dangling_outputs", Args{"ord": Folder, "depth": 2})
	if got := len(objects(want["dangling"])); got != 1 {
		t.Errorf("the existing check reports %d", got)
	}
	// over the component cap the scan is not judged at all
	out := r.Srv.OK("n4_commissioning_report", Args{"scope": Folder, "sections": []any{"links"}, "max_components": 1})
	s = shxSection(t, out, "links")
	if why, _ := comCheck(t, s, "dangling_outputs")["not_checked"].(string); !strings.Contains(why, "narrow the scope") || out["truncated"] != true {
		t.Errorf("over the cap: %v", s)
	}
}

var comNamingTables = map[string][]string{
	"points": {"slot:/Logic/A,A,Supply Temp,control:NumericPoint,0", "slot:/Logic/B,B,Supply Temp,control:NumericPoint,0",
		"slot:/Other/C,C,Supply Temp,control:NumericPoint,0", "slot:/Logic/Zone$20Temp,Zone Temp,Zone Temp,control:NumericPoint,0",
		"slot:/Logic/NumericWritable,NumericWritable,NumericWritable,control:NumericWritable,0",
		"slot:/Logic/BooleanPoint1,BooleanPoint1,BooleanPoint1,control:BooleanPoint,0",
		"slot:/Logic/Thing2,Thing2,Thing2,vendor:Thing,0", "slot:/Logic/Thing2x,Thing2x,Thing2x,vendor:Thing,0"},
}

func TestCommissioningListsDuplicateDisplayNamesAmongSiblings(t *testing.T) {
	r, _ := comRig(t, comNamingTables)
	s := comSection(t, r, "naming", nil)
	c := comWant(t, s, "duplicate_display_names", 2, "slot:/Logic/A", "slot:/Logic/B")
	if fact := comFact(c, "slot:/Logic/B"); fact != `display name "Supply Temp" is shared by 2 siblings` {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningListsNamesEscapedInTheSlotPath(t *testing.T) {
	r, _ := comRig(t, comNamingTables)
	c := comWant(t, comSection(t, r, "naming", nil), "escaped_names", 1, "slot:/Logic/Zone$20Temp")
	if fact := comFact(c, "slot:/Logic/Zone$20Temp"); fact != "slot name Zone$20Temp holds escaped characters ($20)" {
		t.Errorf("fact = %q", fact)
	}
}

func TestCommissioningListsPointsStillCalledByAPaletteDefaultName(t *testing.T) {
	r, _ := comRig(t, comNamingTables)
	comWant(t, comSection(t, r, "naming", nil), "palette_default_names", 3,
		"slot:/Logic/NumericWritable", "slot:/Logic/BooleanPoint1", "slot:/Logic/Thing2")
}
