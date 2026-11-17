package blackbox

// n4_set_slot on a link target (backlog OD001, retro hist-alarms M9): the link writes the slot
// again at its next update, so the dry run lists the links into it and says the value will be
// overwritten. The write is still planned; a slot without links gets an empty list and no note.

import (
	"strings"
	"testing"
)

func TestASetSlotDryRunListsTheLinksIntoTheSlot(t *testing.T) {
	r := verifiedRig(t)
	k := r.AddWithOut("K", 10.0)
	a := r.Add("3", "A", "")
	r.Set(a.Handle, "label", map[string]any{"nm": "p", "t": "baja:String", "v": "x"})
	r.Set(a.Handle, "note", map[string]any{"nm": "p", "t": "baja:String", "v": "y"})
	r.Link(k, "out", a, "label")
	dry := r.Srv.Dry("n4_set_slot", Args{"ord": Folder + "/" + a.Name, "slot": "label", "value": "z", "value_type": "baja:String"})
	p := planOf(dry)
	links := objects(p["incoming_links"])
	if len(links) != 1 || links[0]["source_slot"] != "out" || links[0]["target_slot"] != "label" ||
		links[0]["enabled"] != true || !strings.HasSuffix(links[0]["source_ord"].(string), k.Handle) {
		t.Fatalf("incoming_links = %v", p["incoming_links"])
	}
	if got := noteWith(p["notes"], "incoming_links"); len(got) != 1 || !strings.Contains(got[0], "overwritten") {
		t.Errorf("notes = %v", p["notes"])
	}
	other := planOf(r.Srv.Dry("n4_set_slot", Args{"ord": Folder + "/" + a.Name, "slot": "note", "value": "z", "value_type": "baja:String"}))
	if _, listed := other["incoming_links"]; listed || len(noteWith(other["notes"], "incoming_links")) != 0 {
		t.Errorf("unlinked slot: links %v notes %v", other["incoming_links"], other["notes"])
	}
}

// The token covers the links: one added after the dry run makes the execute refuse.
func TestASetSlotTokenIsRefusedWhenALinkIntoTheSlotAppears(t *testing.T) {
	r := verifiedRig(t)
	k := r.AddWithOut("K", 10.0)
	a := r.Add("3", "A", "")
	r.Set(a.Handle, "label", map[string]any{"nm": "p", "t": "baja:String", "v": "x"})
	args := Args{"ord": Folder + "/" + a.Name, "slot": "label", "value": "z", "value_type": "baja:String"}
	dry := r.Srv.Dry("n4_set_slot", args)
	r.Link(k, "out", a, "label")
	if err := r.Srv.Err("n4_set_slot", Execute(args, dry["confirmation_token"])); !strings.Contains(err, "does not match") {
		t.Errorf("error = %q", err)
	}
}
