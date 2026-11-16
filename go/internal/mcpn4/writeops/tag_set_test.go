package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const tagSvc = "station:|slot:/Services/TagDictionaryService"

// tagLoads is a station with the dictionary Lab (namespace tst: cap a Double, label a String,
// calc a smart tag) and the point P holding tst:cap 1.0, tst:label "a" and a plain slot tst$3axx
// (no metadata flag) whose name a defined tag tst:xx would use.
func tagLoads(t *testing.T, pTree string) *loads {
	t.Helper()
	trees := map[string]string{
		tagSvc: `{"": {"h": "40", "t": "tagdictionary:TagDictionaryService"},
			"Lab": {"n": "Lab", "h": "41", "t": "tagdictionary:TagDictionary"},
			"Lab/namespace": {"n": "namespace", "t": "baja:String", "v": "tst"}}`,
		tagSvc + "/Lab/tagDefinitions": `{"": {"h": "42", "t": "tagdictionary:TagInfoList"},
			"cap": {"n": "cap", "h": "43", "t": "tagdictionary:SimpleTagInfo"},
			"cap/defValue": {"n": "defValue", "t": "baja:Double", "v": "0.0"},
			"label": {"n": "label", "h": "44", "t": "tagdictionary:SimpleTagInfo"},
			"label/defValue": {"n": "defValue", "t": "baja:String", "v": ""},
			"calc": {"n": "calc", "h": "45", "t": "tagdictionary:SmartTagInfo"},
			"xx": {"n": "xx", "h": "46", "t": "tagdictionary:SimpleTagInfo"},
			"xx/defValue": {"n": "defValue", "t": "baja:String", "v": ""},
			"box": {"n": "box", "h": "47", "t": "tagdictionary:SimpleTagInfo"},
			"mode": {"n": "mode", "h": "48", "t": "tagdictionary:DynamicEnumTagInfo"}}`,
		"station:|slot:/P": pTree,
	}
	all := pyjson.NewObject()
	for o, tree := range trees {
		n, err := pyjson.Unmarshal([]byte(tree))
		if err != nil {
			t.Fatal(o, err)
		}
		all.Set(o, n)
	}
	return &loads{nodes: all}
}

const tagP = `{"": {"h": "6", "t": "control:NumericWritable"},
	"tst$3acap": {"nm": "p", "n": "tst$3acap", "f": "m", "t": "baja:Double", "v": "1.0"},
	"tst$3alabel": {"nm": "p", "n": "tst$3alabel", "f": "m", "t": "baja:String", "v": "a"},
	"tst$3axx": {"nm": "p", "n": "tst$3axx", "t": "baja:String", "v": "z"},
	"zz$3aq": {"nm": "p", "n": "zz$3aq", "f": "m", "t": "baja:String", "v": "s"}}`

func tagPlan(t *testing.T, add ...any) (*plan.Planned, error) {
	t.Helper()
	return writeops.SetTags.Plan(tagLoads(t, tagP), obj("ords", []any{"station:|slot:/P"}, "add", add))
}

func TestSetTagsPlanRefusesWhatTheDefinitionsForbid(t *testing.T) {
	for _, c := range []struct {
		add  *pyjson.Object
		want string
	}{
		{obj("id", "tst:calc"), "smart tag"},
		{obj("id", "tst:xx", "value", "q"), "not a direct tag"},
		{obj("id", "tst:cap", "value", "w"), "finite number"},
		{obj("id", "tst:box", "value", "w"), "has no value"},
		{obj("id", "tst:mode"), "value is required"},
	} {
		if _, err := tagPlan(t, c.add); err == nil || !strings.Contains(plan.PyStr(err), c.want) {
			t.Errorf("%v: %v (want %q)", c.add, err, c.want)
		}
	}
}

func TestSetTagsPlanSetsAChangedValueAndSkipsAnEqualOne(t *testing.T) {
	p, err := tagPlan(t, obj("id", "tst:cap", "value", pyjson.IntLiteral("2")), obj("id", "tst:label", "value", "a"))
	if err != nil {
		t.Fatal(err)
	}
	if got := marshal(t, p.Ops); got != `[{"nm": "s", "h": "6", "n": "tst$3acap", "b": {"nm": "p", "t": "baja:Double", "v": "2.0"}}]` {
		t.Errorf("ops %s", got)
	}
	if got := marshal(t, p.Inverse); got != `[{"nm": "s", "h": "6", "n": "tst$3acap", "b": {"nm": "p", "t": "baja:Double", "v": "1.0"}}]` {
		t.Errorf("inverse %s", got)
	}
	if got := marshal(t, p.Notes); !strings.Contains(got, "tst:label on station:|slot:/P already holds that value") {
		t.Errorf("notes %s", got)
	}
	if _, err := tagPlan(t, obj("id", "tst:label", "value", "a")); err == nil || !strings.Contains(plan.PyStr(err), "nothing to change") {
		t.Errorf("all equal: %v", err)
	}
}

func TestSetTagsPlanBoundsTheBatch(t *testing.T) {
	add := []any{}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		add = append(add, obj("id", "zz:"+n))
	}
	ords := []any{}
	for i := 0; i < 41; i++ {
		ords = append(ords, "station:|slot:/P"+strings.Repeat("x", i))
	}
	_, err := writeops.SetTags.Plan(tagLoads(t, tagP), obj("ords", ords, "add", add, "allow_ad_hoc", true))
	if err == nil || !strings.Contains(plan.PyStr(err), "205 ops") {
		t.Errorf("41 x 5: %v", err)
	}
}

func TestSetTagsReadbackReportsANormalizedValue(t *testing.T) {
	p, err := tagPlan(t, obj("id", "tst:cap", "value", pyjson.IntLiteral("2")))
	if err != nil {
		t.Fatal(err)
	}
	after := strings.Replace(tagP, `"v": "1.0"`, `"v": "2.00"`, 1)
	_, _, observed, verdict, err := writeops.SetTags.Readback(tagLoads(t, after), nil, p, []any{}, nil)
	if err != nil || verdict != "verified" {
		t.Fatalf("verdict %s err %v", verdict, err)
	}
	if got := marshal(t, observed); !strings.Contains(got, `"coerced": [{"id": "tst:cap", "sent": "2.0", "stored": "2.00"}]`) {
		t.Errorf("observed %s", got)
	}
	wrong := strings.Replace(tagP, `"v": "1.0"`, `"v": "3.0"`, 1)
	if _, _, _, verdict, _ := writeops.SetTags.Readback(tagLoads(t, wrong), nil, p, []any{}, nil); verdict != "mismatch" {
		t.Errorf("another value: %s", verdict)
	}
}

func TestSetTagsPlanRefusesBadArguments(t *testing.T) {
	st := tagLoads(t, tagP)
	p := []any{"station:|slot:/P"}
	for _, c := range []struct {
		args *pyjson.Object
		want string
	}{
		{obj("ords", []any{pyjson.IntLiteral("5")}, "remove", []any{"tst:cap"}), "ORD strings"},
		{obj("ords", []any{"station:|slot:/P", "station:|slot:/P"}, "remove", []any{"tst:cap"}), "twice"},
		{obj("ords", p, "add", []any{obj("id", "tst:cap"), obj("id", "tst:cap")}), "add names tst:cap twice"},
		{obj("ords", p, "remove", []any{"tst:xx"}), "no direct tag tst:xx"},
		{obj("ords", p, "add", []any{obj("id", "zz:q", "value", pyjson.IntLiteral("1"))}, "allow_ad_hoc", true), "changing a tag's type"},
		{obj("ords", p, "add", []any{obj("id", "tst:label", "type", "baja:Double", "value", pyjson.IntLiteral("1"))}), "defined as baja:String"},
	} {
		if _, err := writeops.SetTags.Plan(st, c.args); err == nil || !strings.Contains(plan.PyStr(err), c.want) {
			t.Errorf("%v: %v (want %q)", c.args, err, c.want)
		}
	}
}

func TestSetTagsReadbackSeesAWrongTypeAndATagStillThere(t *testing.T) {
	st := tagLoads(t, tagP)
	p, err := writeops.SetTags.Plan(st, obj("ords", []any{"station:|slot:/P"}, "add", []any{obj("id", "tst:box")},
		"remove", []any{"tst:label"}))
	if err != nil {
		t.Fatal(err)
	}
	added := strings.Replace(tagP, `"tst$3alabel": {"nm": "p", "n": "tst$3alabel", "f": "m", "t": "baja:String", "v": "a"},`,
		`"tst$3abox": {"nm": "p", "n": "tst$3abox", "f": "m", "t": "baja:Marker", "v": "M"},`, 1)
	if _, _, _, v, _ := writeops.SetTags.Readback(tagLoads(t, added), nil, p, []any{}, nil); v != "verified" {
		t.Fatalf("applied: %s", v)
	}
	labelKept := strings.Replace(tagP, `"tst$3axx"`, `"tst$3abox": {"nm": "p", "n": "tst$3abox", "f": "m", "t": "baja:Marker", "v": "M"}, "tst$3axx"`, 1)
	wrongType := strings.Replace(added, `"t": "baja:Marker"`, `"t": "baja:String"`, 1)
	for name, tree := range map[string]string{"label kept": labelKept, "wrong type": wrongType} {
		if _, _, _, v, _ := writeops.SetTags.Readback(tagLoads(t, tree), nil, p, []any{}, nil); v != "mismatch" {
			t.Errorf("%s: %s", name, v)
		}
	}
}
