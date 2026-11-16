package readtools_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// neqlServer is a connected server whose station answers GET /ord/ with routes.
func neqlServer(t *testing.T, routes ...readtoolstest.OrdRoute) (*readtoolstest.Server, *readtoolstest.Station) {
	t.Helper()
	st := readtoolstest.NewStation()
	st.OrdRoutes = routes
	s := readtoolstest.NewServer(st, session.Config{})
	s.Connect(t)
	return s, st
}

func marshal(t *testing.T, v any) string {
	t.Helper()
	b, err := pyjson.Marshal(v, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// The measured 4.15 answer: a BOM, one Ord column, local:|station:| ORDs.
const neqlBody = "\xef\xbb\xbfOrd\nlocal:|station:|slot:/ImplTAG/Pt1\nlocal:|station:|slot:/ImplTAG/Pt2\n"

func TestNeqlQueryListsTheMatchingComponents(t *testing.T) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Body: neqlBody})
	text, isErr, out := s.Call(t, "n4_neql_query", "base_ord", "station:|slot:/ImplTAG", "neql", ` tst:label = "hello" `)
	if isErr {
		t.Fatal(text)
	}
	if got := st.OrdRequests[0].Ord; got != `station:|slot:/ImplTAG|neql:tst:label = "hello"|view:file:ITableToCsv` {
		t.Errorf("ord %s", got)
	}
	for k, want := range map[string]string{
		"ords":      `["station:|slot:/ImplTAG/Pt1", "station:|slot:/ImplTAG/Pt2"]`,
		"count":     `2`,
		"truncated": `false`,
		"neql":      `"tst:label = \"hello\""`,
	} {
		v, _ := out.Get(k)
		if got := marshal(t, v); got != want {
			t.Errorf("%s = %s, want %s", k, got, want)
		}
	}
}

func TestNeqlQueryWithBqlReturnsItsColumns(t *testing.T) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Body: "\xef\xbb\xbfSlot Path,Value\nslot:/ImplTAG/Pt1,\"0,00\"\n"})
	text, _, out := s.Call(t, "n4_neql_query", "neql", "n:point", "bql", "select slotPath, out.value")
	if len(st.OrdRequests) == 0 {
		t.Fatal(text)
	}
	if got := st.OrdRequests[0].Ord; got != "station:|slot:/|neql:n:point|bql:select slotPath, out.value|view:file:ITableToCsv" {
		t.Errorf("ord %s", got)
	}
	cols, _ := out.Get("columns")
	rows, _ := out.Get("rows")
	if marshal(t, cols) != `["Slot Path", "Value"]` || marshal(t, rows) != `[{"Slot Path": "slot:/ImplTAG/Pt1", "Value": "0,00"}]` {
		t.Errorf("columns %s rows %s", marshal(t, cols), marshal(t, rows))
	}
}

func TestNeqlQueryCapsRows(t *testing.T) {
	s, _ := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Body: neqlBody})
	_, _, out := s.Call(t, "n4_neql_query", "neql", "n:point", "max_rows", pyjson.IntLiteral("1"))
	ords, _ := out.Get("ords")
	tr, _ := out.Get("truncated")
	if len(ords.([]any)) != 1 || tr != true {
		t.Errorf("ords %v truncated %v", ords, tr)
	}
}

func TestNeqlQueryRefusals(t *testing.T) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Code: 400})
	for _, kv := range [][]any{
		{"neql", "a | b"}, {"neql", "  "}, {"neql", "n:point\x01"}, {"neql", "n:point", "base_ord", "h:2"},
		{"neql", "n:point", "bql", "delete"}, {"neql", "n:point", "bql", "select a from b"},
	} {
		if text, isErr, _ := s.Call(t, "n4_neql_query", kv...); !isErr {
			t.Errorf("%v accepted: %s", kv, text)
		}
	}
	if len(st.OrdRequests) != 0 {
		t.Errorf("a refused query reached the station: %v", st.OrdRequests)
	}
	text, isErr, _ := s.Call(t, "n4_neql_query", "neql", "tst:label =")
	if !isErr || !strings.Contains(text, "tst:label =") || !strings.Contains(text, "NEQL") {
		t.Errorf("syntax error: %s", text)
	}
}
