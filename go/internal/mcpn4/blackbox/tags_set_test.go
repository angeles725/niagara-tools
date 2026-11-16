package blackbox

// n4_set_tags over the real BOX client: tags validated against the installed dictionaries,
// added, changed and removed as metadata slots, read back and rolled back.

import (
	"fmt"
	"strings"
	"testing"
)

// tagRig is a writes-allowed rig whose station holds a Lab dictionary (namespace tst: cap a
// Double defaulting to 2.0, label a String, box a Marker) and a Niagara dictionary whose load
// omits its namespace (the type declares n), plus the point Pt under the Folder.
func tagRig(t *testing.T) (*dRig, Component) {
	d := destructiveRig(t, nil)
	svc := d.Add("2", "Services", "baja:Folder")
	tds := d.Add(svc.Handle, "TagDictionaryService", "tagdictionary:TagDictionaryService")
	lab := d.Add(tds.Handle, "Lab", "tagdictionary:TagDictionary")
	d.Set(lab.Handle, "namespace", map[string]any{"nm": "p", "t": "baja:String", "v": "tst"})
	defs := d.Add(lab.Handle, "tagDefinitions", "tagdictionary:TagInfoList")
	for name, dv := range map[string]map[string]any{
		"cap":   {"nm": "p", "t": "baja:Double", "v": "2.0"},
		"label": {"nm": "p", "t": "baja:String", "v": ""},
		"box":   nil,
	} {
		info := d.Add(defs.Handle, name, "tagdictionary:SimpleTagInfo")
		if dv != nil {
			d.Set(info.Handle, "defValue", dv)
		}
	}
	d.Add(lab.Handle, "tagGroupDefinitions", "tagdictionary:TagGroupInfoList")
	d.Add(lab.Handle, "relationDefinitions", "tagdictionary:RelationInfoList")
	nia := d.Add(tds.Handle, "Niagara", "tagdictionary:NiagaraTagDictionary")
	ndefs := d.Add(nia.Handle, "tagDefinitions", "tagdictionary:TagInfoList")
	d.Add(ndefs.Handle, "point", "tagdictionary:SimpleTagInfo")
	d.Station.SetRegistry(map[string]any{"tagdictionary:NiagaraTagDictionary": map[string]any{"c": []any{
		map[string]any{"st": "p", "n": "namespace", "v": map[string]any{"t": "baja:String", "v": "n"}}}}})
	return d, d.Add("3", "Pt", "control:NumericWritable")
}

func TestSetTagsAddsChangesAndRemovesTagsWithRollback(t *testing.T) {
	d, pt := tagRig(t)
	pord := Folder + "/Pt"
	if _, err := d.Station.AddSlot(pt.Handle, "tst$3alabel", map[string]any{"nm": "p", "t": "baja:String", "v": "old"}, "m"); err != nil {
		t.Fatal(err)
	}
	args := Args{"ords": []any{pord}, "add": []any{
		map[string]any{"id": "tst:cap", "value": 4.5}, map[string]any{"id": "tst:box"},
		map[string]any{"id": "n:point"}, map[string]any{"id": "tst:label", "value": "new"}}}
	plan := d.Srv.Dry("n4_set_tags", args)
	if ops, _ := plan["plan"].(map[string]any)["ops"].([]any); len(ops) != 4 {
		t.Fatalf("plan %v", plan)
	}
	res := d.Srv.OK("n4_set_tags", Execute(args, plan["confirmation_token"]))
	if res["verdict"] != "verified" {
		t.Fatalf("verdict %v: %v", res["verdict"], res["observed"])
	}
	got := map[string]string{}
	for _, c := range d.node("Pt").Children {
		if strings.Contains(c.Name, "$3a") {
			got[c.Name] = c.Type + "=" + fmt.Sprint(c.Value) + "/" + c.Flags
		}
	}
	want := map[string]string{"tst$3acap": "baja:Double=4.5/m", "tst$3abox": "baja:Marker=M/m",
		"n$3apoint": "baja:Marker=M/m", "tst$3alabel": "baja:String=new/m"}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q (all %v)", k, got[k], v, got)
		}
	}
	rb := d.rollback(res["batch_id"])
	if rb["verdict"] != "verified" {
		t.Fatalf("rollback %v", rb)
	}
	for _, c := range d.node("Pt").Children {
		if c.Name == "tst$3alabel" && c.Value != any("old") || c.Name == "tst$3acap" || c.Name == "tst$3abox" || c.Name == "n$3apoint" {
			t.Errorf("after rollback %s = %s", c.Name, c.Value)
		}
	}
	rm := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{pord}, "remove": []any{"tst:label"}})
	if rm["verdict"] != "verified" || d.node("Pt", "tst$3alabel") != nil {
		t.Fatalf("remove %v", rm)
	}
	if rb := d.rollback(rm["batch_id"]); rb["verdict"] != "verified" || d.node("Pt", "tst$3alabel") == nil {
		t.Fatalf("remove rollback %v", rb)
	}
}

func TestSetTagsRefusals(t *testing.T) {
	d, pt := tagRig(t)
	pord := Folder + "/Pt"
	if _, err := d.Station.AddSlot(pt.Handle, "tst$3acap", map[string]any{"nm": "p", "t": "baja:Double", "v": "1.0"}, "m"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		args Args
		want string
	}{
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "tst:nope"}}}, "not defined"},
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "tst:nope"}}, "allow_ad_hoc": true}, "installed"},
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "zz:free"}}}, "allow_ad_hoc"},
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "tst:cap", "value": "x"}}}, "finite number"},
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "tst:cap", "type": "baja:String", "value": "x"}}}, "baja:Double"},
		{Args{"ords": []any{pord}, "add": []any{map[string]any{"id": "nocolon"}}}, "ns:name"},
		{Args{"ords": []any{pord}, "remove": []any{"tst:label"}}, "no direct tag"},
		{Args{"ords": []any{pord}, "remove": []any{"out"}}, "ns:name"},
		{Args{"ords": []any{pord}}, "nothing"},
		{Args{"ords": []any{"station:|slot:/Elsewhere"}, "remove": []any{"tst:cap"}}, "scope"},
	} {
		if msg := d.Srv.Err("n4_set_tags", c.args); !strings.Contains(msg, c.want) {
			t.Errorf("%v: %s (want %q)", c.args, msg, c.want)
		}
	}
	free := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{pord}, "allow_ad_hoc": true,
		"add": []any{map[string]any{"id": "zz:free", "value": "x"}}})
	if free["verdict"] != "verified" || d.node("Pt", "zz$3afree") == nil {
		t.Fatalf("ad hoc in a namespace no dictionary installs: %v", free)
	}
}
