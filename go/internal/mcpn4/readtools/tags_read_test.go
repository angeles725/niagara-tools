package readtools_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// taggedPoint is a point as a 4.15 station loads it: two direct tags (one with flags rhm), a
// plain slot, a metadata slot that is transient (no tag), a tag group and a custom relation.
const taggedPoint = `{"": {"h": "6", "t": "control:NumericWritable"},
	"out": {"n": "out", "t": "baja:StatusNumeric"},
	"hs$3apoint": {"nm": "p", "n": "hs$3apoint", "f": "m", "t": "baja:Marker", "v": "M"},
	"tst$3alabel": {"nm": "p", "n": "tst$3alabel", "f": "rhm", "t": "baja:String", "v": "hello"},
	"scratch": {"nm": "p", "n": "scratch", "f": "mt", "t": "baja:String", "v": "x"},
	"r": {"nm": "p", "n": "r", "f": "r", "t": "baja:Relation"},
	"r/relationId": {"nm": "p", "n": "relationId", "v": "n:tagGroup"},
	"r/sourceOrd": {"nm": "p", "n": "sourceOrd", "v": "slot:/Services/TagDictionaryService/D/tagGroupDefinitions/ahu"},
	"r1": {"nm": "p", "n": "r1", "t": "baja:Relation"},
	"r1/relationId": {"nm": "p", "n": "relationId", "v": "tst:feeds"},
	"r1/sourceOrd": {"nm": "p", "n": "sourceOrd", "v": "h:5"}}`

func TestReadTagsDirectTagsGroupsAndRelations(t *testing.T) {
	s := serve(t, "station:|slot:/P", taggedPoint)
	text, isErr, out := s.Call(t, "n4_read_tags", "ord", "station:|slot:/P")
	if isErr {
		t.Fatalf("n4_read_tags: %s", text)
	}
	comps, _ := out.Get("components")
	c := comps.([]any)[0].(*pyjson.Object)
	got := func(k string) string {
		v, _ := c.Get(k)
		b, _ := pyjson.Marshal(v, pyjson.Options{})
		return string(b)
	}
	for k, want := range map[string]string{
		"ord":    `"station:|slot:/P"`,
		"handle": `"6"`,
		"direct": `[{"id": "hs:point", "namespace": "hs", "name": "point", "type": "baja:Marker", "value": "M", ` +
			`"slot": "hs$3apoint", "flags": "m"}, {"id": "tst:label", "namespace": "tst", "name": "label", ` +
			`"type": "baja:String", "value": "hello", "slot": "tst$3alabel", "flags": "rhm"}]`,
		"tag_groups": `[{"group_ord": "slot:/Services/TagDictionaryService/D/tagGroupDefinitions/ahu", "slot": "r"}]`,
		"relations":  `{"direct": [{"id": "tst:feeds", "endpoint_ord": "h:5", "inbound": false, "slot": "r1"}]}`,
		// this test client cannot call the session handler: said, never an empty list
		"implied": `null`,
	} {
		if g := got(k); g != want {
			t.Errorf("%s\n got %s\nwant %s", k, g, want)
		}
	}
	if note := got("implied_unavailable"); note == "null" {
		t.Error("no implied_unavailable note")
	}
}

func TestReadTagsRefusesNoOrdAndANonStationOrd(t *testing.T) {
	s := serve(t, "station:|slot:/P", taggedPoint)
	for _, kv := range [][]any{{}, {"ord", "slot:/P"}, {"ords", []any{"station:|slot:/P", "h:2"}}} {
		if text, isErr, _ := s.Call(t, "n4_read_tags", kv...); !isErr {
			t.Errorf("%v accepted: %s", kv, text)
		}
	}
}

func TestReadTagsWithoutImpliedSaysNothingAboutThem(t *testing.T) {
	s := serve(t, "station:|slot:/P", taggedPoint)
	_, _, out := s.Call(t, "n4_read_tags", "ord", "station:|slot:/P", "implied", false)
	comps, _ := out.Get("components")
	c := comps.([]any)[0].(*pyjson.Object)
	for _, k := range []string{"implied", "implied_unavailable"} {
		if v, has := c.Get(k); has {
			t.Errorf("%s = %v with implied false", k, v)
		}
	}
}
