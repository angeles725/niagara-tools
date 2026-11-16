package blackbox

// n4_set_tags for tag groups and relations: a group is an n:tagGroup relation (flag r) to the
// group's definition; a relation an outbound relation to the endpoint's handle ORD.

import (
	"encoding/json"
	"strings"
	"testing"
)

// groupRig is tagRig plus the Lab group tst:pump (implying tst:box) and the relation tst:feeds,
// and a second point Q the relation points at.
func groupRig(t *testing.T) (*dRig, Component, Component) {
	d, pt := tagRig(t)
	var groupsH, relsH string
	for _, c := range d.nodeFrom("2", "Services", "TagDictionaryService", "Lab").Children {
		switch c.Name {
		case "tagGroupDefinitions":
			groupsH = c.Handle
		case "relationDefinitions":
			relsH = c.Handle
		}
	}
	g := d.Add(groupsH, "pump", "tagdictionary:TagGroupInfo")
	list := d.Add(g.Handle, "tagList", "tagdictionary:TagInfoList")
	d.Add(list.Handle, "box", "tagdictionary:SimpleTagInfo")
	d.Add(relsH, "feeds", "tagdictionary:RelationInfo")
	d.Add(relsH, "drives", "tagdictionary:RelationInfo")
	d.Add("3", "R", "control:NumericWritable")
	return d, pt, d.Add("3", "Q", "control:NumericWritable")
}

func relations(d *dRig) map[string]string {
	out := map[string]string{}
	for _, c := range d.node("Pt").Children {
		if c.Type == "baja:Relation" {
			fields := map[string]string{}
			for _, f := range c.Children {
				fields[f.Name], _ = f.Value.(string)
			}
			out[c.Name] = fields["relationId"] + " -> " + fields["sourceOrd"] + " /" + c.Flags
		}
	}
	return out
}

func TestSetTagsAddsAndRemovesGroupsAndRelationsWithRollback(t *testing.T) {
	d, _, q := groupRig(t)
	pord := Folder + "/Pt"
	args := Args{"ords": []any{pord}, "add_groups": []any{"tst:pump"},
		"add_relations": []any{map[string]any{"id": "tst:feeds", "endpoint_ord": Folder + "/Q"}}}
	res := d.Srv.RunWrite("n4_set_tags", args)
	if res["verdict"] != "verified" {
		t.Fatalf("add: %v", res)
	}
	want := map[string]string{"r": "n:tagGroup -> slot:/Services/TagDictionaryService/Lab/tagGroupDefinitions/pump /r",
		"r1": "tst:feeds -> h:" + q.Handle + " /"}
	if got := relations(d); len(got) != 2 || got["r"] != want["r"] || got["r1"] != want["r1"] {
		t.Fatalf("relations %v, want %v", got, want)
	}
	b, _ := json.Marshal(res["inverse"])
	if s := string(b); !strings.Contains(s, `"n":"r1"`) || !strings.Contains(s, `"n":"r"`) || strings.Contains(s, `r?`) {
		t.Errorf("inverse names %s", s)
	}
	again := d.Srv.Err("n4_set_tags", args)
	if !strings.Contains(again, "already there") {
		t.Errorf("repeat: %s", again)
	}
	if rb := d.rollback(res["batch_id"]); rb["verdict"] != "verified" || len(relations(d)) != 0 {
		t.Fatalf("rollback %v, relations %v", rb, relations(d))
	}
	d.Srv.RunWrite("n4_set_tags", args)
	rm := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{pord}, "remove_groups": []any{"tst:pump"},
		"remove_relations": []any{map[string]any{"id": "tst:feeds", "endpoint_ord": Folder + "/Q"}}})
	if rm["verdict"] != "verified" || len(relations(d)) != 0 {
		t.Fatalf("remove %v, relations %v", rm, relations(d))
	}
	if rb := d.rollback(rm["batch_id"]); rb["verdict"] != "verified" || relations(d)["r"] != want["r"] {
		t.Fatalf("remove rollback %v, relations %v", rb, relations(d))
	}
}

func TestSetTagsGroupAndRelationRefusals(t *testing.T) {
	d, _, _ := groupRig(t)
	pord := Folder + "/Pt"
	for _, c := range []struct {
		args Args
		want string
	}{
		{Args{"ords": []any{pord}, "add_groups": []any{"tst:nope"}}, "no installed dictionary defines the tag group tst:nope"},
		{Args{"ords": []any{pord}, "add_groups": []any{"tst:pump"}, "remove_groups": []any{"tst:pump"}}, "both added and removed"},
		{Args{"ords": []any{pord}, "remove_groups": []any{"tst:pump"}}, "has no tag group tst:pump"},
		{Args{"ords": []any{pord}, "add_relations": []any{map[string]any{"id": "tst:other", "endpoint_ord": Folder + "/Q"}}}, "not defined by the installed dictionary Lab"},
		{Args{"ords": []any{pord}, "add_relations": []any{map[string]any{"id": "zz:rel", "endpoint_ord": Folder + "/Q"}}}, "allow_ad_hoc"},
		{Args{"ords": []any{pord}, "add_relations": []any{map[string]any{"id": "n:tagGroup", "endpoint_ord": Folder + "/Q"}}}, "add_groups"},
		{Args{"ords": []any{pord}, "add_relations": []any{map[string]any{"id": "tst:feeds", "endpoint_ord": Folder + "/Missing"}}}, "Missing"},
	} {
		if msg := d.Srv.Err("n4_set_tags", c.args); !strings.Contains(msg, c.want) {
			t.Errorf("%v: %s (want %q)", c.args, msg, c.want)
		}
	}
}

// A relation is matched by id and endpoint together: another id to the same endpoint, or the
// same id to another endpoint, is a new relation.
func TestSetTagsMatchesARelationByIdAndEndpoint(t *testing.T) {
	d, _, _ := groupRig(t)
	pord := Folder + "/Pt"
	rel := func(id, end string) Args {
		return Args{"ords": []any{pord}, "add_relations": []any{map[string]any{"id": id, "endpoint_ord": Folder + "/" + end}}}
	}
	d.Srv.RunWrite("n4_set_tags", rel("tst:feeds", "Q"))
	for _, a := range []Args{rel("tst:drives", "Q"), rel("tst:feeds", "R")} {
		if res := d.Srv.RunWrite("n4_set_tags", a); res["verdict"] != "verified" {
			t.Errorf("%v: %v", a, res)
		}
	}
	if got := relations(d); len(got) != 3 {
		t.Errorf("relations %v", got)
	}
}
