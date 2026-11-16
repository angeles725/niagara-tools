package blackbox

// Outgoing links through remove and rollback (bench retro 2026-11-15 delta 20): the remove
// records the links that leave the removed component, n4_rollback re-creates them and its
// read-back compares them.

import (
	"strings"
	"testing"
)

// removeAcknowledged removes the component name under parent with acknowledge_links and returns
// the dry run's plan and the execution reply.
func (d *dRig) removeAcknowledged(parent, name string) (plan, out map[string]any) {
	d.T.Helper()
	args := acknowledged(Args{"parent_ord": parent, "name": name})
	reply := d.Srv.Dry("n4_remove_component", args)
	return planOf(reply), d.Srv.OK("n4_remove_component", Execute(args, reply["confirmation_token"]))
}

// outgoingEntries are the outgoing_link entries of an inverse.
func outgoingEntries(inverse any) []map[string]any {
	var out []map[string]any
	for _, e := range objects(inverse) {
		if spec, ok := e["outgoing_link"].(map[string]any); ok {
			out = append(out, spec)
		}
	}
	return out
}

// fedBy reports whether c holds exactly one link and it reads out of the component now at
// Folder/<src> (out -> in10).
func (d *dRig) fedBy(c Component, src string) bool {
	n := d.node(src)
	l := d.links(c)
	return n != nil && len(l) == 1 && l[0] == [3]any{"h:" + n.Handle, "out", "in10"}
}

func TestTheRemoveInverseRecordsEachOutgoingLink(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, b := d.linkedConst()
	plan, _ := d.removeAcknowledged(Folder, k.Name)
	got := outgoingEntries(plan["inverse"])
	if len(got) != 2 {
		t.Fatalf("inverse = %v", plan["inverse"])
	}
	for i, c := range []Component{a, b} {
		e := got[i]
		if e["target"] != Folder+"/"+c.Name || e["target_h"] != c.Handle || e["target_slot"] != "in10" ||
			e["source_path"] != "" || e["source_slot"] != "out" {
			t.Errorf("entry %d = %v", i, e)
		}
	}
	if len(noteWith(plan["notes"], "2 link(s)", "n4_rollback re-creates them")) != 1 {
		t.Errorf("notes = %v", plan["notes"])
	}
}

func TestARollbackReCreatesTheOutgoingLinksAndReadsVerified(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, b := d.linkedConst()
	_, out := d.removeAcknowledged(Folder, k.Name)
	back := d.rollback(out["batch_id"])
	if back["verdict"] != "verified" {
		t.Errorf("verdict = %v: %v", back["verdict"], back)
	}
	if !d.fedBy(a, k.Name) || !d.fedBy(b, k.Name) {
		t.Errorf("links after the rollback: A %v, B %v", d.links(a), d.links(b))
	}
	if !same(Dig(back, "observed", "outgoing_links"), map[string]any{"restored": 2, "missing": []any{}}) {
		t.Errorf("observed = %v", back["observed"])
	}
}

func TestARollbackWhoseStationDropsAnOutgoingLinkReadsMismatch(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, b := d.linkedConst()
	_, out := d.removeAcknowledged(Folder, k.Name)
	d.f.set(func(f *faults) { // the station accepts B's link and keeps none
		f.replace = func(sck string, arg map[string]any) (any, error, bool) {
			if sck != "checkLinks" || arg["t"] != b.Handle {
				return nil, nil, false
			}
			return []any{[]any{map[string]any{"v": true, "r": nil, "s": "Link"}}}, nil, true
		}
	})
	back := d.rollback(out["batch_id"])
	if back["verdict"] != "mismatch" || !d.fedBy(a, k.Name) || len(d.links(b)) != 0 {
		t.Errorf("verdict = %v, A %v, B %v", back["verdict"], d.links(a), d.links(b))
	}
	missing := objects(Dig(back, "observed", "outgoing_links", "missing"))
	if len(missing) != 1 || missing[0]["target"] != Folder+"/"+b.Name {
		t.Errorf("observed = %v", back["observed"])
	}
}

func TestARollbackRefusesAnOutgoingLinkWhoseTargetLeftTheWriteScope(t *testing.T) {
	d := destructiveRig(t, nil)
	area := d.Add("3", "Area", "baja:Folder")
	k := d.Add(area.Handle, "K", "")
	d.Set(k.Handle, "out", StatusNumeric(10.0, "0"))
	a := d.Add("3", "A", "")
	d.Link(k, "out", a, "in10")
	areaOrd := Folder + "/" + area.Name
	_, out := d.removeAcknowledged(areaOrd, k.Name)
	d.restart(func(o *Options) { o.WriteScopes = []string{areaOrd} })
	text := d.Srv.Err("n4_rollback", Args{"batch_id": out["batch_id"]})
	if !strings.Contains(text, Folder+"/"+a.Name) || !strings.Contains(text, "outside the write scope") {
		t.Errorf("text = %s", text)
	}
	if d.nodeFrom(area.Handle, k.Name) != nil {
		t.Errorf("the refused rollback re-created K")
	}
}
