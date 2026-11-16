package blackbox

// n4_set_tags probes of the independent verification (2026-11-15): scope before any request,
// argument refusals, exact rollback of removed tags and relations, the change guard.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// verifyTagRig is tagRig plus the Lab group ahu (implying cap), the relation tst:feeds, the point
// Q, and on Pt: tst:cap 1.0 (flags mh), plain slots tst$3axx and zz$3aplain, the String tag zz:q.
func verifyTagRig(t *testing.T) (*dRig, Component) {
	d, pt := tagRig(t)
	var gdefs, rdefs string
	d.Station.Do(func(st *boxstation.Station) {
		lab := st.ByHandle("2").Child("Services").Child("TagDictionaryService").Child("Lab")
		gdefs, rdefs = lab.Child("tagGroupDefinitions").Handle, lab.Child("relationDefinitions").Handle
	})
	g := d.Add(gdefs, "ahu", "tagdictionary:TagGroupInfo")
	d.Add(d.Add(g.Handle, "tagList", "tagdictionary:TagInfoList").Handle, "cap", "tagdictionary:SimpleTagInfo")
	d.Add(rdefs, "feeds", "tagdictionary:RelationInfo")
	for _, s := range []struct{ name, typ, v, flags string }{
		{"tst$3acap", "baja:Double", "1.0", "mh"}, {"tst$3axx", "baja:String", "z", ""},
		{"zz$3aplain", "baja:String", "s", ""}, {"zz$3aq", "baja:String", "s", "m"},
	} {
		if _, err := d.Station.AddSlot(pt.Handle, s.name, map[string]any{"nm": "p", "t": s.typ, "v": s.v}, s.flags); err != nil {
			t.Fatal(err)
		}
	}
	d.Add("3", "Q", "control:NumericWritable")
	return d, pt
}

func TestSetTagsRefusesOutOfScopeTargetsBeforeAnyRequest(t *testing.T) {
	d, _ := verifyTagRig(t)
	for name, ords := range map[string][]any{
		"outside": {"station:|slot:/Elsewhere"}, "mixed": {Folder + "/Pt", "station:|slot:/Elsewhere"},
		"sibling prefix": {"station:|slot:/Folder2/Pt"}, "chained": {Folder + "/Pt|bql:select *"},
	} {
		before := d.Station.Requests()
		if msg := d.Srv.Err("n4_set_tags", Args{"ords": ords, "remove": []any{"tst:cap"}}); !strings.Contains(msg, "scope") && !strings.Contains(msg, "plain") {
			t.Errorf("%s: %s", name, msg)
		}
		if n := d.Station.Requests() - before; n != 0 {
			t.Errorf("%s: %d requests before the refusal", name, n)
		}
	}
}

func TestSetTagsTokenOfAWiderScopeIsRefusedAfterANarrowerRestart(t *testing.T) {
	d, _ := verifyTagRig(t)
	args := Args{"ords": []any{Folder + "/Pt"}, "remove": []any{"tst:cap"}}
	tok := d.Srv.Dry("n4_set_tags", args)["confirmation_token"]
	d.restart(func(o *Options) { o.WriteScopes = []string{Folder + "/Q"} })
	if msg := d.Srv.Err("n4_set_tags", Execute(args, tok)); !strings.Contains(msg, "scope") || d.node("Pt", "tst$3acap") == nil {
		t.Errorf("narrow restart: %s", msg)
	}
}

func TestSetTagsBoundsTargetsAndOps(t *testing.T) {
	d, _ := verifyTagRig(t)
	ords, add := []any{}, []any{}
	for i := 0; i < 51; i++ {
		ords = append(ords, fmt.Sprintf("%s/P%d", Folder, i))
	}
	for i := 0; i < 201; i++ {
		add = append(add, map[string]any{"id": fmt.Sprintf("zz:t%d", i)})
	}
	for want, args := range map[string]Args{
		"1 to 50":  {"ords": ords, "remove": []any{"tst:cap"}},
		"201 ops":  {"ords": []any{Folder + "/Pt"}, "add": add, "allow_ad_hoc": true},
		"1 to 50 ": {"ords": []any{}, "remove": []any{"tst:cap"}},
	} {
		if msg := d.Srv.Err("n4_set_tags", args); !strings.Contains(msg, strings.TrimSpace(want)) {
			t.Errorf("%s: %s", want, msg)
		}
	}
}

func TestSetTagsRefusesInvalidRequests(t *testing.T) {
	d, _ := verifyTagRig(t)
	p, q := []any{Folder + "/Pt"}, Folder+"/Q"
	add := func(e map[string]any) Args { return Args{"ords": p, "add": []any{e}, "allow_ad_hoc": true} }
	cases := map[string]Args{
		"installed dictionary":   add(map[string]any{"id": "n:nope"}),
		"finite number":          add(map[string]any{"id": "tst:cap", "value": true}),
		"a string":               add(map[string]any{"id": "tst:label", "value": 3}),
		"no value":               add(map[string]any{"id": "tst:box", "value": false}),
		"changing a tag's type":  add(map[string]any{"id": "zz:q", "value": 1}),
		"not a direct tag":       add(map[string]any{"id": "zz:plain", "value": "q"}),
		"no direct tag":          {"ords": p, "remove": []any{"zz:plain"}},
		"tag group":              {"ords": p, "add_groups": []any{"tst:nogroup"}},
		"add_groups":             {"ords": p, "add_relations": []any{map[string]any{"id": "n:tagGroup", "endpoint_ord": q}}},
		"namespace of its own":   {"ords": p, "add_relations": []any{map[string]any{"id": "tst:nope", "endpoint_ord": q}}, "allow_ad_hoc": true},
		"defined as baja:Double": {"ords": p, "add": []any{map[string]any{"id": "tst:cap", "type": "baja:String", "value": "1"}}},
	}
	for want, args := range cases {
		if msg := d.Srv.Err("n4_set_tags", args); !strings.Contains(msg, want) {
			t.Errorf("want %q: %s", want, msg)
		}
	}
	for _, bad := range []string{"nocolon", "a:b:c", ":x", "x:", "a|b:c", "a b:c", "", "1a:b"} {
		for _, args := range []Args{add(map[string]any{"id": bad}), {"ords": p, "remove": []any{bad}}} {
			if msg := d.Srv.Err("n4_set_tags", args); !strings.Contains(msg, "ns:name") {
				t.Errorf("id %q: %s", bad, msg)
			}
		}
	}
}

// Blockers B1-B3, B5 of the verification: each request is refused at plan, before any write.
func TestSetTagsRefusesDuplicatesContradictionsAndNonSimpleTypes(t *testing.T) {
	d, _ := verifyTagRig(t)
	p, rel := []any{Folder + "/Pt"}, map[string]any{"id": "tst:feeds", "endpoint_ord": Folder + "/Q"}
	for name, args := range map[string]Args{
		"remove twice":      {"ords": p, "remove": []any{"tst:cap", "tst:cap"}},
		"group twice":       {"ords": p, "add_groups": []any{"tst:ahu", "tst:ahu"}},
		"relation twice":    {"ords": p, "add_relations": []any{rel, rel}},
		"add and remove":    {"ords": p, "add": []any{map[string]any{"id": "tst:cap", "value": 5}}, "remove": []any{"tst:cap"}},
		"component type":    {"ords": p, "allow_ad_hoc": true, "add": []any{map[string]any{"id": "zz:c", "type": "baja:Folder", "value": "x"}}},
		"struct type":       {"ords": p, "allow_ad_hoc": true, "add": []any{map[string]any{"id": "zz:r", "type": "baja:Relation", "value": "x"}}},
		"unknown type text": {"ords": p, "allow_ad_hoc": true, "add": []any{map[string]any{"id": "zz:n", "type": "nonsense", "value": "x"}}},
	} {
		if r := d.Srv.Call("n4_set_tags", args); !r.IsError {
			t.Errorf("%s: planned instead of refused", name)
		}
	}
}

func TestSetTagsRemoveRollbackRestoresTypeValueAndFlags(t *testing.T) {
	d, _ := verifyTagRig(t)
	rm := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{Folder + "/Pt"}, "remove": []any{"tst:cap"}})
	if rb := d.rollback(rm["batch_id"]); rm["verdict"] != "verified" || rb["verdict"] != "verified" {
		t.Fatalf("remove %v rollback %v", rm["verdict"], rb["verdict"])
	}
	if n := d.node("Pt", "tst$3acap"); n == nil || n.Type != "baja:Double" || n.Value != any("1.0") || n.Flags != "mh" {
		t.Errorf("restored %+v", n)
	}
}

func TestSetTagsRollbackRefusesATagChangedAfterTheBatch(t *testing.T) {
	d, pt := verifyTagRig(t)
	ch := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{Folder + "/Pt"}, "add": []any{map[string]any{"id": "tst:cap", "value": 7}}})
	if err := d.Station.Set(pt.Handle, "tst$3acap", map[string]any{"nm": "p", "t": "baja:Double", "v": "9.0"}); err != nil {
		t.Fatal(err)
	}
	if msg := d.Srv.Err("n4_rollback", Args{"batch_id": ch["batch_id"]}); !strings.Contains(msg, "changed since") {
		t.Errorf("guard: %s", msg)
	}
}

// Blocker B4: a removed relation comes back with the sourceOrd it held, not a rewritten form.
func TestSetTagsRelationRemoveRollbackKeepsTheStoredEndpoint(t *testing.T) {
	d, pt := verifyTagRig(t)
	if _, err := d.Station.AddSlot(pt.Handle, "r?", map[string]any{"nm": "p", "t": "baja:Relation", "s": []any{
		map[string]any{"nm": "p", "n": "relationId", "t": "baja:String", "v": "tst:feeds"},
		map[string]any{"nm": "p", "n": "sourceOrd", "t": "baja:Ord", "v": "slot:/Folder/Q"}}}, ""); err != nil {
		t.Fatal(err)
	}
	rm := d.Srv.RunWrite("n4_set_tags", Args{"ords": []any{Folder + "/Pt"},
		"remove_relations": []any{map[string]any{"id": "tst:feeds", "endpoint_ord": Folder + "/Q"}}})
	if rb := d.rollback(rm["batch_id"]); rm["verdict"] != "verified" || rb["verdict"] != "verified" {
		t.Fatalf("remove %v rollback %v", rm["verdict"], rb["verdict"])
	}
	if src := d.node("Pt", "r", "sourceOrd"); src == nil || src.Value != any("slot:/Folder/Q") {
		t.Errorf("restored sourceOrd %+v", src)
	}
}
