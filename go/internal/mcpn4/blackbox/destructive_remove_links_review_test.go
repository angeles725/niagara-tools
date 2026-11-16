package blackbox

// The native review of the outgoing-link scan (bench retro 2026-11-15 deltas 3 and 20):
// link_scan_ord only narrows, the rollback's budget refusal names the outgoing links, and a scope
// root too large to scan refuses (the deep internal link is pinned in remove_links_internal_test).

import (
	"strconv"
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

func TestTheRollbackBudgetRefusalNamesTheOutgoingLinks(t *testing.T) {
	d := destructiveRig(t, nil)
	k, _, _ := d.linkedConst()
	_, out := d.removeAcknowledged(Folder, k.Name)
	d.restart(func(o *Options) { o.MaxWrites = 2 })
	text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	if !strings.Contains(text, "needs 3 write(s) (1 batch + 0 component(s) + 0 relink(s) + 2 outgoing link(s)) but only 2 remain") {
		t.Errorf("text = %s", text)
	}
	if d.node(k.Name) != nil {
		t.Errorf("the refused rollback re-created K")
	}
}

func TestAScopeRootTooLargeToScanRefusesAndLinkScanOrdNarrowsIt(t *testing.T) {
	d := destructiveRig(t, nil)
	big := d.Add("3", "Big", "baja:Folder")
	for i := 0; i <= 5000; i++ {
		d.Add(big.Handle, "F"+strconv.Itoa(i), "baja:Folder")
	}
	area := d.Add("3", "Area", "baja:Folder")
	k := d.Add("3", "K", "")
	text := d.Srv.Err("n4_remove_component", removeArgs(k.Name))
	for _, want := range []string{"scope root " + Folder, "too large to scan", "5000", "link_scan_ord", "refusing"} {
		if !strings.Contains(text, want) {
			t.Errorf("the refusal misses %q: %s", want, text)
		}
	}
	args := Args{"parent_ord": Folder, "name": k.Name, "link_scan_ord": Folder + "/" + area.Name}
	reply := d.Srv.Dry("n4_remove_component", args)
	d.Srv.OK("n4_remove_component", Execute(args, reply["confirmation_token"]))
	if d.node(k.Name) != nil {
		t.Errorf("K is still there")
	}
}
