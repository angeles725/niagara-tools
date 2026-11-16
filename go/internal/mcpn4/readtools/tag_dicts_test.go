package readtools_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

const dictRows = "\xef\xbb\xbfSlot Path,Namespace,Enabled,Frozen,Version,Status\n" +
	"slot:/Services/TagDictionaryService/Niagara,n,true,true,1.5,{ok}\n" +
	"slot:/Services/TagDictionaryService/Lab,tst,true,false,,{ok}\n"

const dictD = "station:|slot:/Services/TagDictionaryService/"

// dictServer serves the two dictionaries: Niagara (one Marker tag at its class default, one
// String tag) and Lab (one Double tag, one group implying it, one relation).
func dictServer(t *testing.T) (*readtoolstest.Server, *readtoolstest.Station) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "tagdictionary:TagDictionary", Body: dictRows})
	trees := map[string]string{
		dictD + "Niagara/tagDefinitions": `{"": {"h": "1", "t": "tagdictionary:TagInfoList"},
			"device": {"n": "device", "t": "tagdictionary:SimpleTagInfo"},
			"device/validity": {"n": "validity", "t": "tagdictionary:IsTypeCondition", "d": "Is driver:Device"},
			"name": {"n": "name", "t": "tagdictionary:SimpleTagInfo"},
			"name/defValue": {"n": "defValue", "t": "baja:String", "v": ""}}`,
		dictD + "Niagara/tagGroupDefinitions": `{"": {"h": "2", "t": "tagdictionary:TagGroupInfoList"}}`,
		dictD + "Niagara/relationDefinitions": `{"": {"h": "3", "t": "tagdictionary:RelationInfoList"},
			"parent": {"n": "parent", "t": "tagdictionary:RelationInfo"}}`,
		dictD + "Lab/tagDefinitions": `{"": {"h": "4", "t": "tagdictionary:TagInfoList"},
			"cap": {"n": "cap", "t": "tagdictionary:SimpleTagInfo"},
			"cap/defValue": {"n": "defValue", "t": "baja:Double", "v": "0.0"}}`,
		dictD + "Lab/tagGroupDefinitions": `{"": {"h": "5", "t": "tagdictionary:TagGroupInfoList"},
			"box": {"n": "box", "t": "tagdictionary:TagGroupInfo"},
			"box/tagList": {"n": "tagList", "t": "tagdictionary:TagInfoList"},
			"box/tagList/cap": {"n": "cap", "t": "tagdictionary:SimpleTagInfo"}}`,
		dictD + "Lab/relationDefinitions": `{"": {"h": "6", "t": "tagdictionary:RelationInfoList"},
			"feeds": {"n": "feeds", "t": "tagdictionary:RelationInfo"}}`,
	}
	s.Station.Trees = map[string]*pyjson.Object{}
	for o, tree := range trees {
		n, err := pyjson.Unmarshal([]byte(tree))
		if err != nil {
			t.Fatal(o, err)
		}
		s.Station.Trees[o] = n.(*pyjson.Object)
	}
	return s, st
}

func TestListTagDictionariesGivesDefinitionsGroupsAndRelations(t *testing.T) {
	s, st := dictServer(t)
	text, isErr, out := s.Call(t, "n4_list_tag_dictionaries")
	if isErr {
		t.Fatal(text)
	}
	if !strings.Contains(st.OrdRequests[0].Ord, "station:|slot:/Services/TagDictionaryService|bql:select slotPath, namespace") {
		t.Errorf("ord %s", st.OrdRequests[0].Ord)
	}
	d, _ := out.Get("dictionaries")
	want := `[{"name": "Niagara", "ord": "station:|slot:/Services/TagDictionaryService/Niagara", "namespace": "n", ` +
		`"enabled": true, "frozen": true, "version": "1.5", "status": "{ok}", "tags": [` +
		`{"id": "n:device", "info": "tagdictionary:SimpleTagInfo", "type": "baja:Marker", "default": null, "validity": "Is driver:Device"}, ` +
		`{"id": "n:name", "info": "tagdictionary:SimpleTagInfo", "type": "baja:String", "default": "", "validity": null}], ` +
		`"tag_groups": [], "relations": ["n:parent"]}, ` +
		`{"name": "Lab", "ord": "station:|slot:/Services/TagDictionaryService/Lab", "namespace": "tst", ` +
		`"enabled": true, "frozen": false, "version": "", "status": "{ok}", "tags": [` +
		`{"id": "tst:cap", "info": "tagdictionary:SimpleTagInfo", "type": "baja:Double", "default": "0.0", "validity": null}], ` +
		`"tag_groups": [{"id": "tst:box", "group_ord": "slot:/Services/TagDictionaryService/Lab/tagGroupDefinitions/box", "tags": ["tst:cap"]}], ` +
		`"relations": ["tst:feeds"]}]`
	if got := marshal(t, d); got != want {
		t.Errorf("dictionaries\n got %s\nwant %s", got, want)
	}
	if tr, _ := out.Get("truncated"); tr != false {
		t.Errorf("truncated %v", tr)
	}
}

func TestListTagDictionariesFiltersAndBounds(t *testing.T) {
	s, _ := dictServer(t)
	_, _, out := s.Call(t, "n4_list_tag_dictionaries", "dictionary", "tst")
	d, _ := out.Get("dictionaries")
	if n := len(d.([]any)); n != 1 || !strings.Contains(marshal(t, d), `"name": "Lab"`) {
		t.Errorf("namespace filter: %s", marshal(t, d))
	}
	_, _, out = s.Call(t, "n4_list_tag_dictionaries", "dictionary", "Niagara", "max_items", pyjson.IntLiteral("1"))
	d, _ = out.Get("dictionaries")
	tr, _ := out.Get("truncated")
	if got := marshal(t, d); tr != true || strings.Contains(got, "n:name") || strings.Contains(got, "n:parent") {
		t.Errorf("max_items 1: truncated %v %s", tr, got)
	}
	if text, isErr, _ := s.Call(t, "n4_list_tag_dictionaries", "dictionary", "nope"); !isErr || !strings.Contains(text, "Niagara (n)") {
		t.Errorf("unknown dictionary: %s", text)
	}
}
