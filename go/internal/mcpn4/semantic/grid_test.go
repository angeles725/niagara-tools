package semantic

import (
	"encoding/json"
	"testing"
)

func TestGridExportsHsTagsAndRelationsAsHayson(t *testing.T) {
	ahu := Entity{Ord: "station:|slot:/Drivers/AHU1", Dis: "AHU1",
		Tags: []Tag{{"hs:equip", "baja:Marker", "M"}, {"hs:ahu", "baja:Marker", "M"}, {"n:name", "baja:String", "AHU1"}}}
	sat := Entity{Ord: "station:|slot:/Drivers/AHU1/SAT", Dis: "SAT",
		Tags: []Tag{{"hs:point", "baja:Marker", "M"}, {"hs:temp", "baja:Marker", "M"},
			{"hs:maxVal", "baja:Double", "40.5"}, {"hs:enabled", "baja:Boolean", "true"},
			{"hs:kind", "baja:String", "Number"}, {"hs:tz", "baja:TimeZone", "America/Mexico_City"}},
		Relations: []Relation{{"hs:equipRef", "station:|slot:/Drivers/AHU1"}, {"hs:siteRef", "station:|slot:/Site"}}}
	grid, skipped := Grid([]Entity{ahu, sat})
	got, _ := json.Marshal(grid)
	want := `{"_kind":"grid","cols":[{"name":"id"},{"name":"dis"},{"name":"ahu"},{"name":"enabled"},{"name":"equip"},` +
		`{"name":"equipRef"},{"name":"kind"},{"name":"maxVal"},{"name":"point"},{"name":"siteRef"},{"name":"temp"}],` +
		`"meta":{"ver":"3.0"},"rows":[{"ahu":{"_kind":"marker"},"dis":"AHU1","equip":{"_kind":"marker"},` +
		`"id":{"_kind":"ref","dis":"AHU1","val":"Drivers.AHU1"}},{"dis":"SAT","enabled":true,` +
		`"equipRef":{"_kind":"ref","val":"Drivers.AHU1"},"id":{"_kind":"ref","dis":"SAT","val":"Drivers.AHU1.SAT"},` +
		`"kind":"Number","maxVal":{"_kind":"number","val":40.5},"point":{"_kind":"marker"},` +
		`"siteRef":{"_kind":"ref","val":"Site"},"temp":{"_kind":"marker"}}]}`
	if string(got) != want {
		t.Errorf("grid\n%s\nwant\n%s", got, want)
	}
	if len(skipped) != 1 || skipped[0].Tag != "hs:tz" || skipped[0].Ord != sat.Ord {
		t.Errorf("skipped %+v", skipped)
	}
}

func TestRefIDEscapesAndDisambiguates(t *testing.T) {
	if got := RefID("station:|slot:/Drivers/Room$20A/Temp"); got != "Drivers.Room_20A.Temp" {
		t.Errorf("%s", got)
	}
	if got := RefID("station:|slot:/"); got != "root" {
		t.Errorf("%s", got)
	}
	ids := uniqueIDs([]Entity{{Ord: "station:|slot:/a$20b"}, {Ord: "station:|slot:/a$2Cb"}})
	if ids[0] != "a_20b" || ids[1] != "a_2Cb" {
		t.Errorf("%v", ids)
	}
	ids = uniqueIDs([]Entity{{Ord: "station:|slot:/a b"}, {Ord: "station:|slot:/a?b"}})
	if ids[0] != "a_b" || ids[1] != "a_b~2" {
		t.Errorf("%v", ids)
	}
}

func TestValueRefusesWhatItCannotConvert(t *testing.T) {
	if _, why := value(Tag{"hs:x", "baja:Double", "abc"}); why != "value abc is not a baja:Double" {
		t.Errorf("%s", why)
	}
}
