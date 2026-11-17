package blackbox

// station-audits S10 (bulk tagging) is n4_set_tags itself: N points, add and remove, one dry
// run and one token. This pins it.

import (
	"testing"
)

func TestSetTagsAddsAndRemovesOnSeveralPointsUnderOneToken(t *testing.T) {
	d, pt := tagRig(t)
	pt2 := d.Add("3", "Pt2", "control:NumericWritable")
	label := map[string]any{"nm": "p", "t": "baja:String", "v": "old"}
	for _, h := range []string{pt.Handle, pt2.Handle} {
		if _, err := d.Station.AddSlot(h, "tst$3alabel", label, "m"); err != nil {
			t.Fatal(err)
		}
	}
	args := Args{"ords": []any{Folder + "/Pt", Folder + "/Pt2"}, "add": []any{map[string]any{"id": "tst:box"}},
		"remove": []any{"tst:label"}}
	plan := d.Srv.Dry("n4_set_tags", args)
	if ops, _ := plan["plan"].(map[string]any)["ops"].([]any); len(ops) != 4 {
		t.Fatalf("plan %v", plan)
	}
	res := d.Srv.OK("n4_set_tags", Execute(args, plan["confirmation_token"]))
	if res["verdict"] != "verified" {
		t.Fatalf("verdict %v: %v", res["verdict"], res["observed"])
	}
	for _, name := range []string{"Pt", "Pt2"} {
		if d.node(name, "tst$3abox") == nil || d.node(name, "tst$3alabel") != nil {
			t.Errorf("%s not tagged as planned", name)
		}
	}
	if rb := d.rollback(res["batch_id"]); rb["verdict"] != "verified" || d.node("Pt2", "tst$3alabel") == nil ||
		d.node("Pt", "tst$3abox") != nil {
		t.Fatalf("rollback %v", rb)
	}
}
