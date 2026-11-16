package blackbox

// Edge cases of the outgoing-link scan and its rollback (bench retro 2026-11-15 deltas 3 and 20),
// from the independent verification.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestAnOutgoingLinkFromANestedChildIsListedAndRestored(t *testing.T) {
	d := destructiveRig(t, nil)
	grp := d.Add("3", "Grp", "baja:Folder")
	inner := d.Add(grp.Handle, "Inner", "baja:Folder")
	src := d.Add(inner.Handle, "Src", "")
	d.Set(src.Handle, "out", StatusNumeric(3.0, "0"))
	tgt := d.Add("3", "T", "")
	d.Link(src, "out", tgt, "in10")
	if text := d.Srv.Err("n4_remove_component", removeArgs(grp.Name)); !strings.Contains(text, Folder+"/T (in10 <- Inner/Src/out)") {
		t.Errorf("refusal = %s", text)
	}
	_, out := d.removeAcknowledged(Folder, grp.Name)
	back := d.rollback(out["batch_id"])
	n, l := d.node(grp.Name, "Inner", "Src"), d.links(tgt)
	if back["verdict"] != "verified" || n == nil || len(l) != 1 || l[0] != [3]any{"h:" + n.Handle, "out", "in10"} {
		t.Errorf("verdict %v, links %v", back["verdict"], l)
	}
}

func TestTwoSourcesIntoOneTargetSlotAreBothRestored(t *testing.T) {
	d := destructiveRig(t, nil)
	grp := d.Add("3", "Grp", "baja:Folder")
	s1, s2 := d.Add(grp.Handle, "S1", ""), d.Add(grp.Handle, "S2", "")
	d.Set(s1.Handle, "out", StatusNumeric(1.0, "0"))
	d.Set(s2.Handle, "out", StatusNumeric(2.0, "0"))
	tgt := d.Add("3", "T", "")
	d.Link(s1, "out", tgt, "in10")
	d.Link(s2, "out", tgt, "in10")
	plan, out := d.removeAcknowledged(Folder, grp.Name)
	if len(objects(plan["outgoing_links_broken"])) != 2 {
		t.Errorf("outgoing_links_broken = %v", plan["outgoing_links_broken"])
	}
	back := d.rollback(out["batch_id"])
	if back["verdict"] != "verified" || len(d.links(tgt)) != 2 ||
		!same(Dig(back, "observed", "outgoing_links"), map[string]any{"restored": 2, "missing": []any{}}) {
		t.Errorf("verdict %v, observed %v, links %v", back["verdict"], back["observed"], d.links(tgt))
	}
}

func TestANonBooleanAcknowledgementIsRejected(t *testing.T) {
	d := destructiveRig(t, nil)
	k, _, _ := d.linkedConst()
	for _, v := range []any{"true", 1, nil} {
		args := removeArgs(k.Name)
		args["acknowledge_links"] = v
		if d.Srv.CallError("n4_remove_component", args) == 0 {
			t.Errorf("acknowledge_links %v was accepted", v)
		}
	}
	args := removeArgs(k.Name)
	args["acknowledge_links"] = false
	if text := d.Srv.Err("n4_remove_component", args); !strings.Contains(text, "acknowledge_links: true") {
		t.Errorf("false = %s", text)
	}
	if d.node(k.Name) == nil {
		t.Errorf("K was removed")
	}
}

func TestATokenGoesStaleWhenTheOutgoingLinksChange(t *testing.T) {
	d := destructiveRig(t, nil)
	k, a, _ := d.linkedConst()
	args := acknowledged(removeArgs(k.Name))
	reply := d.Srv.Dry("n4_remove_component", args)
	d.Link(k, "out", d.Add("3", "C", ""), "in10")
	if text := d.Srv.Err("n4_remove_component", Execute(args, reply["confirmation_token"])); !strings.Contains(text, "run the dry run again") {
		t.Errorf("text = %s", text)
	}
	plain := removeArgs("Lone")
	lone := d.AddWithOut("Lone", 1.0)
	token := d.Srv.Dry("n4_remove_component", plain)["confirmation_token"]
	d.Link(lone, "out", d.Add("3", "X", ""), "in10")
	if text := d.Srv.Err("n4_remove_component", Execute(plain, token)); !strings.Contains(text, "acknowledge_links: true") {
		t.Errorf("text = %s", text)
	}
	if d.node(k.Name) == nil || d.node("Lone") == nil || len(d.links(a)) != 1 {
		t.Errorf("a stale token removed something")
	}
}

func TestARollbackReportsAnOutgoingTargetThatIsGoneOrReplaced(t *testing.T) {
	for _, replace := range []bool{false, true} {
		d := destructiveRig(t, nil)
		k, a, b := d.linkedConst()
		_, out := d.removeAcknowledged(Folder, k.Name)
		d.Station.Do(func(st *boxstation.Station) { removeChild(st.ByHandle("3"), b.Name) })
		reason := "target not found"
		if replace {
			nb := d.Add("3", b.Name, "")
			reason = "target replaced since the remove (handle " + b.Handle + ", now " + nb.Handle + ")"
		}
		back := d.rollback(out["batch_id"])
		missing := objects(Dig(back, "observed", "outgoing_links", "missing"))
		if back["verdict"] != "mismatch" || !d.fedBy(a, k.Name) || len(missing) != 1 ||
			missing[0]["target"] != Folder+"/"+b.Name || missing[0]["reason"] != reason {
			t.Errorf("replace=%v: verdict %v, observed %v", replace, back["verdict"], back["observed"])
		}
	}
}

func TestARollbackUnderANarrowerScopeHoldingBothEndsVerifies(t *testing.T) {
	d := destructiveRig(t, nil)
	area := d.Add("3", "Area", "baja:Folder")
	k := d.Add(area.Handle, "K", "")
	d.Set(k.Handle, "out", StatusNumeric(10.0, "0"))
	a := d.Add("3", "A", "")
	d.Link(k, "out", a, "in10")
	areaOrd := Folder + "/" + area.Name
	_, out := d.removeAcknowledged(areaOrd, k.Name)
	d.restart(func(o *Options) { o.WriteScopes = []string{areaOrd, Folder + "/" + a.Name} })
	if back := d.rollback(out["batch_id"]); back["verdict"] != "verified" {
		t.Errorf("verdict %v, observed %v", back["verdict"], back["observed"])
	}
}
