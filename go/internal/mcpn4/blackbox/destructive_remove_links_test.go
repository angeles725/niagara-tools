package blackbox

// Links that leave a removed component (bench retro 2026-11-15 delta 3): the remove scans the
// write scope that holds the parent by default, refuses until the caller acknowledges the links
// it found, and link_scan_ord narrows that scan.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// linkedConst is the bench reproduction: a NumericConst K (out 10.0) under the Folder whose out
// feeds A.in10 and B.in10, two blocks outside it.
func (d *dRig) linkedConst() (k, a, b Component) {
	d.T.Helper()
	k = d.AddWithOut("K", 10.0)
	a, b = d.Add("3", "A", ""), d.Add("3", "B", "")
	d.Link(k, "out", a, "in10")
	d.Link(k, "out", b, "in10")
	return k, a, b
}

// links are the links a component holds, as (sourceOrd, sourceSlotName, targetSlotName).
func (d *dRig) links(c Component) [][3]any {
	var out [][3]any
	n := d.nodeFrom(c.Handle)
	d.Station.Do(func(*boxstation.Station) {
		for _, l := range n.Children {
			if l.Type != "baja:Link" {
				continue
			}
			var row [3]any
			for i, slot := range []string{"sourceOrd", "sourceSlotName", "targetSlotName"} {
				if s := l.Child(slot); s != nil {
					row[i] = s.Value
				}
			}
			out = append(out, row)
		}
	})
	return out
}

func acknowledged(args Args) Args {
	args["acknowledge_links"] = true
	return args
}

func TestRemovingALinkSourceRefusesUntilItsLinksAreAcknowledged(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, b := d.linkedConst()
	text := d.Srv.Err("n4_remove_component", removeArgs(k.Name))
	for _, want := range []string{Folder + "/" + a.Name + " (in10 <- out)", Folder + "/" + b.Name + " (in10 <- out)",
		"acknowledge_links", "2 link(s)", "scanned " + Folder} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal misses %q: %s", want, text)
		}
	}
	if d.node(k.Name) == nil || len(d.links(a)) != 1 || len(d.links(b)) != 1 {
		t.Errorf("the refused remove touched the station")
	}
}

func TestAnAcknowledgedRemoveListsTheLinksAndExecutes(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, b := d.linkedConst()
	reply := d.Srv.Dry("n4_remove_component", acknowledged(removeArgs(k.Name)))
	plan := planOf(reply)
	want := []any{
		map[string]any{"target": Folder + "/" + a.Name, "target_slot": "in10", "source_path": "", "source_slot": "out"},
		map[string]any{"target": Folder + "/" + b.Name, "target_slot": "in10", "source_path": "", "source_slot": "out"}}
	if !same(plan["outgoing_links_broken"], want) {
		t.Errorf("outgoing_links_broken = %v", plan["outgoing_links_broken"])
	}
	if len(noteWith(plan["notes"], "scanned "+Folder, "2 link(s)")) != 1 {
		t.Errorf("notes = %v", plan["notes"])
	}
	d.Srv.OK("n4_remove_component", Execute(acknowledged(removeArgs(k.Name)), reply["confirmation_token"]))
	if d.node(k.Name) != nil || len(d.links(a)) != 0 || len(d.links(b)) != 0 {
		t.Errorf("after the remove: K %v, links %v %v", d.node(k.Name), d.links(a), d.links(b))
	}
}

func TestAComponentWithNoOutgoingLinksRemovesWithoutAcknowledgement(t *testing.T) {
	d := destructiveRig(t, nil)
	nn, _ := d.group("Grp")
	reply := d.Srv.Dry("n4_remove_component", removeArgs(nn))
	plan := planOf(reply)
	if _, ok := plan["outgoing_links_broken"]; ok {
		t.Errorf("outgoing_links_broken = %v", plan["outgoing_links_broken"])
	}
	if len(noteWith(plan["notes"], "scanned "+Folder, "no outgoing links")) != 1 {
		t.Errorf("notes = %v", plan["notes"])
	}
	d.Srv.OK("n4_remove_component", Execute(removeArgs(nn), reply["confirmation_token"]))
	if d.node(nn) != nil {
		t.Errorf("%s is still there", nn)
	}
}

func TestLinkScanOrdNarrowsTheScan(t *testing.T) {
	d := destructiveRig(t, nil)
	area := d.Add("3", "Area", "baja:Folder")
	k := d.Add(area.Handle, "K", "")
	d.Set(k.Handle, "out", StatusNumeric(10.0, "0"))
	inside, outside := d.Add(area.Handle, "A", ""), d.Add("3", "B", "")
	d.Link(k, "out", inside, "in10")
	d.Link(k, "out", outside, "in10")
	areaOrd := Folder + "/" + area.Name
	args := Args{"parent_ord": areaOrd, "name": k.Name, "link_scan_ord": areaOrd}
	text := d.Srv.Err("n4_remove_component", args)
	if !strings.Contains(text, areaOrd+"/"+inside.Name+" (in10 <- out)") || strings.Contains(text, Folder+"/"+outside.Name+" ") ||
		!strings.Contains(text, "1 link(s)") || !strings.Contains(text, "stored outside "+areaOrd+" are not detected") {
		t.Errorf("narrowed refusal = %s", text)
	}
	whole := d.Srv.Err("n4_remove_component", Args{"parent_ord": areaOrd, "name": k.Name})
	if !strings.Contains(whole, "2 link(s)") || !strings.Contains(whole, Folder+"/"+outside.Name+" (in10 <- out)") {
		t.Errorf("default refusal = %s", whole)
	}
}
