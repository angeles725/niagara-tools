package readtools_test

import (
	"os"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

const equipTree = `{"": {"h": "5", "t": "baja:Folder"},
	"hs$3aequip": {"nm": "p", "n": "hs$3aequip", "f": "m", "t": "baja:Marker", "v": "M"},
	"hs$3adis": {"nm": "p", "n": "hs$3adis", "f": "m", "t": "baja:String", "v": "Air handler 1"}}`

const pointTree = `{"": {"h": "6", "t": "control:NumericWritable"},
	"hs$3apoint": {"nm": "p", "n": "hs$3apoint", "f": "m", "t": "baja:Marker", "v": "M"},
	"hs$3amaxVal": {"nm": "p", "n": "hs$3amaxVal", "f": "m", "t": "baja:Double", "v": "40.5"},
	"hs$3atz": {"nm": "p", "n": "hs$3atz", "f": "m", "t": "baja:TimeZone", "v": "UTC"},
	"tst$3alabel": {"nm": "p", "n": "tst$3alabel", "f": "m", "t": "baja:String", "v": "hello"},
	"r1": {"nm": "p", "n": "r1", "t": "baja:Relation"},
	"r1/relationId": {"nm": "p", "n": "relationId", "v": "hs:equipRef"},
	"r1/sourceOrd": {"nm": "p", "n": "sourceOrd", "v": "station:|slot:/AHU1"}}`

func semanticServer(t *testing.T) (*readtoolstest.Server, *readtoolstest.Station) {
	t.Helper()
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:",
		Body: "\xef\xbb\xbfOrd\nlocal:|station:|slot:/AHU1\nlocal:|station:|slot:/AHU1/SAT\n"})
	st.Trees = map[string]*pyjson.Object{}
	for ord, tree := range map[string]string{"station:|slot:/AHU1": equipTree, "station:|slot:/AHU1/SAT": pointTree} {
		n, err := pyjson.Unmarshal([]byte(tree))
		if err != nil {
			t.Fatal(err)
		}
		st.Trees[ord] = n.(*pyjson.Object)
	}
	return s, st
}

func TestSemanticExportBuildsAHaystackGrid(t *testing.T) {
	s, st := semanticServer(t)
	text, isErr, out := s.Call(t, "n4_semantic_export")
	if isErr {
		t.Fatal(text)
	}
	if got := st.OrdRequests[0].Ord; got != "station:|slot:/|neql:hs:site or hs:space or hs:equip or hs:point|view:file:ITableToCsv" {
		t.Errorf("ord %s", got)
	}
	want := `{"_kind": "grid", "cols": [{"name": "id"}, {"name": "dis"}, {"name": "equip"}, {"name": "equipRef"}, ` +
		`{"name": "maxVal"}, {"name": "point"}], "meta": {"ver": "3.0"}, "rows": [{"dis": "Air handler 1", ` +
		`"equip": {"_kind": "marker"}, "id": {"_kind": "ref", "dis": "Air handler 1", "val": "AHU1"}}, {"dis": "SAT", ` +
		`"equipRef": {"_kind": "ref", "val": "AHU1"}, "id": {"_kind": "ref", "dis": "SAT", "val": "AHU1.SAT"}, ` +
		`"maxVal": {"_kind": "number", "val": 40.5}, "point": {"_kind": "marker"}}]}`
	if got := marshal(t, exportField(out, "grid")); got != want {
		t.Errorf("grid\n%s\nwant\n%s", got, want)
	}
	if got := marshal(t, exportField(out, "skipped")); got != `[{"ord": "station:|slot:/AHU1/SAT", "tag": "hs:tz", "why": "type baja:TimeZone has no Haystack conversion here"}]` {
		t.Errorf("skipped %s", got)
	}
	if got := marshal(t, exportField(out, "entities")); got != "2" {
		t.Errorf("entities %s", got)
	}
	if _, ok := out.Get("implied_unavailable"); !ok {
		t.Error("the test client cannot read implied tags: the reply must say so")
	}
}

func TestSemanticExportSavesThePrivateFileOnce(t *testing.T) {
	s, _ := semanticServer(t)
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	s.Ctx.StateDir = dir
	text, isErr, out := s.Call(t, "n4_semantic_export", "save_as", "site-a", "implied", false)
	if isErr {
		t.Fatal(text)
	}
	path := pyjson.Str(exportField(out, "path"))
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 || !strings.HasSuffix(path, "/exports/site-a.json") {
		t.Fatalf("%s %v %v", path, info, err)
	}
	if _, has := out.Get("grid"); has {
		t.Error("grid inline with save_as")
	}
	if text, isErr, _ := s.Call(t, "n4_semantic_export", "save_as", "site-a"); !isErr {
		t.Errorf("second save over the file accepted: %s", text)
	}
}
func exportField(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }
