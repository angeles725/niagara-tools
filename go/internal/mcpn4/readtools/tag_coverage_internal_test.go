package readtools

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

func tcJSON(t *testing.T, v any) string {
	t.Helper()
	if list, ok := v.([]*pyjson.Object); ok {
		v = tcCapped(list)
	}
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTcEntityOfReadsPointMarkerAndOutboundEquipRefs(t *testing.T) {
	entry, err := pyjson.Unmarshal([]byte(`{"handle": "9",
		"direct": [{"id": "tst:label"}], "implied": [{"id": "hs:point"}],
		"relations": {"direct": [{"id": "hs:equipRef", "endpoint_ord": "h:5", "inbound": false},
			{"id": "hs:equipRef", "endpoint_ord": "h:7", "inbound": true},
			{"id": "hs:siteRef", "endpoint_ord": "h:1", "inbound": false}],
			"implied": [{"id": "hs:equipRef", "endpoint_ord": "local:|station:|h:6", "inbound": false}]}}`))
	if err != nil {
		t.Fatal(err)
	}
	e := tcEntityOf("station:|slot:/P", entry.(*pyjson.Object))
	want := tcEntity{ord: "station:|slot:/P", handle: "9", direct: []string{"tst:label"}, point: true,
		equipRefs: []string{"h:5", "local:|station:|h:6"}}
	if !reflect.DeepEqual(e, want) {
		t.Errorf("%+v", e)
	}
}

func TestTcPointGapsNamesWhatEachPointMisses(t *testing.T) {
	gaps, noPoint, noEquip := tcPointGaps([]tcEntity{
		{ord: "covered", point: true, equipRefs: []string{"h:5"}},
		{ord: "bare"},
		{ord: "no-equip", point: true},
		{ord: "no-marker", equipRefs: []string{"h:5"}},
	})
	want := `[{"point": "bare", "missing": ["hs:point", "hs:equipRef"]}, {"point": "no-equip", "missing": ["hs:equipRef"]}, ` +
		`{"point": "no-marker", "missing": ["hs:point"]}]`
	if got := tcJSON(t, gaps); got != want || noPoint != 2 || noEquip != 2 {
		t.Errorf("%s %d %d", got, noPoint, noEquip)
	}
}

func TestTcOrphanEquipsMatchesHandleAndSlotEndpoints(t *testing.T) {
	points := []tcEntity{{equipRefs: []string{"local:|station:|h:5"}}, {equipRefs: []string{"station:|slot:/B"}}}
	equips := []tcEntity{{ord: "station:|slot:/A", handle: "5"}, {ord: "station:|slot:/B", handle: "6"},
		{ord: "station:|slot:/C", handle: "7"}}
	if got := tcJSON(t, tcOrphanEquips(points, equips)); got != `[{"equip": "station:|slot:/C"}]` {
		t.Error(got)
	}
}

func tcLabDicts(only string) tcDicts {
	return tcDicts{installed: map[string]string{"tst": "Lab", "n": "Niagara"}, only: only,
		defined: func(id tags.ID) (bool, error) { return id.String() == "tst:cap" || id.String() == "n:point", nil }}
}

func TestTcUnknownTagsCountsUndefinedAndGroupsUninstalledNamespaces(t *testing.T) {
	ents := []tcEntity{
		{ord: "A", direct: []string{"tst:cap", "tst:bogus", "hs:point", "hs:siteRef"}},
		{ord: "B", direct: []string{"tst:bogus", "hs:temp", "n:point"}},
	}
	got, err := tcUnknownTags(ents, tcLabDicts(""))
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"tag": "hs:*", "count": 3, "example": "A", "why": "no dictionary installed for namespace 'hs' (ad hoc tags)"}, ` +
		`{"tag": "tst:bogus", "count": 2, "example": "A", "why": "not defined by the installed dictionary Lab"}]`
	if s := tcJSON(t, got); s != want {
		t.Errorf("%s", s)
	}
	only, _ := tcUnknownTags(ents, tcLabDicts("tst"))
	if s := tcJSON(t, only); s != `[{"tag": "tst:bogus", "count": 2, "example": "A", "why": "not defined by the installed dictionary Lab"}]` {
		t.Errorf("dictionary tst: %s", s)
	}
}

func TestTcCappedKeepsAtMostTheItemBound(t *testing.T) {
	list := []*pyjson.Object{}
	for i := 0; i < tcMaxItems+5; i++ {
		list = append(list, obj("i", i))
	}
	if got := len(tcCapped(list)); got != tcMaxItems {
		t.Errorf("%d", got)
	}
}
