package blackbox

// n4_read_tags over the real BOX client: direct tags from the load, implied tags and relations
// from the session handler.

import (
	"reflect"
	"testing"
)

// tagged is a Folder child with a marker tag, a string tag, a tag group and implied tags.
func tagged(t *testing.T, r *Rig) Component {
	t.Helper()
	c := r.Add("3", "Pt", "")
	for _, s := range []struct {
		name  string
		body  map[string]any
		flags string
	}{
		{"hs$3apoint", map[string]any{"nm": "p", "t": "baja:Marker", "v": "M"}, "m"},
		{"tst$3alabel", map[string]any{"nm": "p", "t": "baja:String", "v": "hello"}, "m"},
		{"r?", map[string]any{"nm": "p", "t": "baja:Relation", "s": []any{
			map[string]any{"nm": "p", "n": "relationId", "t": "baja:String", "v": "n:tagGroup"},
			map[string]any{"nm": "p", "n": "sourceOrd", "t": "baja:Ord", "v": "slot:/G/ahu"}}}, "r"},
	} {
		if _, err := r.Station.AddSlot(c.Handle, s.name, s.body, s.flags); err != nil {
			t.Fatal(err)
		}
	}
	r.Station.SetImplied(c.Handle, map[string]any{"hs:ahuM": nil, "n:nameS": "Pt"},
		[]map[string]any{{"rId": "n:parent", "inbound": false, "entityOrd": "local:|station:|h:3"}})
	return c
}

func TestReadTagsReadsDirectGroupAndImpliedTags(t *testing.T) {
	r := stationRig(t)
	c := tagged(t, r)
	got := objects(r.Srv.OK("n4_read_tags", Args{"ord": Folder + "/Pt", "relations": "all"})["components"])[0]
	if got["handle"] != c.Handle {
		t.Errorf("handle %v", got["handle"])
	}
	ids := []string{}
	for _, d := range objects(got["direct"]) {
		ids = append(ids, d["id"].(string)+"="+d["type"].(string)+"/"+d["flags"].(string))
	}
	if want := []string{"hs:point=baja:Marker/m", "tst:label=baja:String/m"}; !reflect.DeepEqual(ids, want) {
		t.Errorf("direct %v", ids)
	}
	if g := objects(got["tag_groups"]); len(g) != 1 || g[0]["group_ord"] != "slot:/G/ahu" || g[0]["slot"] != "r" {
		t.Errorf("tag_groups %v", g)
	}
	wantImplied := []map[string]any{{"id": "hs:ahu", "type": "baja:Marker", "value": nil},
		{"id": "n:name", "type": "baja:String", "value": "Pt"}}
	if im := objects(got["implied"]); !reflect.DeepEqual(im, wantImplied) {
		t.Errorf("implied %v", im)
	}
	rel := got["relations"].(map[string]any)
	if im := objects(rel["implied"]); len(im) != 1 || im[0]["id"] != "n:parent" || im[0]["endpoint_ord"] != "local:|station:|h:3" {
		t.Errorf("implied relations %v", im)
	}
}

func TestReadTagsWithoutImpliedMakesNoSessionCall(t *testing.T) {
	r := stationRig(t)
	tagged(t, r)
	got := objects(r.Srv.OK("n4_read_tags", Args{"ords": []any{Folder + "/Pt"}, "implied": false, "relations": "none"})["components"])[0]
	if _, has := got["implied"]; has {
		t.Errorf("implied read: %v", got["implied"])
	}
	if _, has := got["relations"]; has {
		t.Errorf("relations read: %v", got["relations"])
	}
}
