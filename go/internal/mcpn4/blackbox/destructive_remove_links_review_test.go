package blackbox

// The native review of the outgoing-link scan (bench retro 2026-11-15 deltas 3 and 20):
// link_scan_ord only narrows and the rollback's budget refusal names the outgoing links (the deep
// internal link and the too-large root are pinned in writeops' remove_links tests).

import (
	"strings"
	"testing"
)

func TestLinkScanOrdOutsideTheDefaultRootRefuses(t *testing.T) {
	d := destructiveRig(t, nil)
	area, other := d.Add("3", "Area", "baja:Folder"), d.Add("3", "Other", "baja:Folder")
	k := d.Add(area.Handle, "K", "")
	areaOrd, otherOrd := Folder+"/"+area.Name, Folder+"/"+other.Name
	d.restart(func(o *Options) { o.WriteScopes = []string{areaOrd, otherOrd} })
	text := d.Srv.Err("n4_remove_component", Args{"parent_ord": areaOrd, "name": k.Name, "link_scan_ord": otherOrd})
	for _, want := range []string{"link_scan_ord " + otherOrd, "not inside " + areaOrd, "only narrows", "refusing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal misses %q: %s", want, text)
		}
	}
	if d.node(area.Name, k.Name) == nil {
		t.Errorf("K was removed")
	}
	reply := d.Srv.Dry("n4_remove_component", Args{"parent_ord": areaOrd, "name": k.Name, "link_scan_ord": areaOrd})
	if reply["confirmation_token"] == nil {
		t.Errorf("link_scan_ord equal to the default root was refused: %v", reply)
	}
}

// The outgoing links a rollback re-creates are outside --max-writes too (retro 2026-11-17 Δ6): a budget
// of 1 that once refused this rollback (it needed 3 writes) now runs it without spending one.
func TestARollbackRecreatesTheOutgoingLinksBeyondTheBudget(t *testing.T) {
	d := destructiveRig(t, nil)
	k, _, _ := d.linkedConst()
	_, out := d.removeAcknowledged(Folder, k.Name)
	d.restart(func(o *Options) { o.MaxWrites = 1 })
	back := d.Srv.RunWrite("n4_rollback", Args{"batch_id": out["batch_id"]})
	obs, _ := back["observed"].(map[string]any)
	if links, _ := obs["outgoing_links"].(map[string]any); d.node(k.Name) == nil || links["restored"] != float64(2) {
		t.Errorf("the rollback did not re-create K and its links: %v", back)
	}
	if b, _ := back["write_budget"].(map[string]any); b["executed"] != float64(0) {
		t.Errorf("write_budget = %v: the rollback must not spend a write", back["write_budget"])
	}
}

// The refusal of a scope root too large to scan is pinned in writeops/remove_links_budget_test.go
// with a lowered budget; here only the narrowing it asks for, which needs no big tree.
func TestLinkScanOrdInsideTheDefaultRootProceeds(t *testing.T) {
	d := destructiveRig(t, nil)
	area := d.Add("3", "Area", "baja:Folder")
	k := d.Add("3", "K", "")
	args := Args{"parent_ord": Folder, "name": k.Name, "link_scan_ord": Folder + "/" + area.Name}
	reply := d.Srv.Dry("n4_remove_component", args)
	if len(noteWith(planOf(reply)["notes"], "scanned "+Folder+"/"+area.Name+" (link_scan_ord)")) != 1 {
		t.Errorf("notes = %v", planOf(reply)["notes"])
	}
	d.Srv.OK("n4_remove_component", Execute(args, reply["confirmation_token"]))
	if d.node(k.Name) != nil {
		t.Errorf("K is still there")
	}
}
