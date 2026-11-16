package readtools_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

func TestVerifyNeqlRefusesBeforeAnyRequest(t *testing.T) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Body: neqlBody})
	for _, kv := range [][]any{
		{"neql", `tst:label = "a|b"`}, {"neql", "n:point\nor n:x"}, {"neql", ""},
		{"neql", "n:point", "bql", "select a|b"}, {"neql", "n:point", "base_ord", "station:|slot:/A|bql:select *"},
		{"neql", strings.Repeat("n:a or ", 400)},
	} {
		if text, isErr, _ := s.Call(t, "n4_neql_query", kv...); !isErr {
			t.Errorf("%v accepted: %.200s", kv, text)
		}
	}
	if len(st.OrdRequests) != 0 {
		t.Errorf("a refused query reached the station: %v", st.OrdRequests)
	}
}

func TestVerifyNeqlEmptyAnswerAndServerError(t *testing.T) {
	s, _ := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Body: "\xef\xbb\xbfOrd\n"})
	text, isErr, out := s.Call(t, "n4_neql_query", "neql", "n:nothing")
	if isErr {
		t.Fatal(text)
	}
	if got := marshal(t, out); !strings.Contains(got, `"ords": []`) || !strings.Contains(got, `"count": 0`) {
		t.Errorf("empty %s", got)
	}
	s2, _ := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Code: 500})
	text, isErr, _ = s2.Call(t, "n4_neql_query", "neql", "n:point")
	t.Logf("HTTP 500: isErr=%v %s", isErr, text)
	if !isErr {
		t.Errorf("500 accepted")
	}
	s3, _ := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Code: 400})
	text, _, _ = s3.Call(t, "n4_neql_query", "neql", "hs:temp >")
	if !strings.Contains(text, `'hs:temp >'`) || !strings.Contains(text, "HTTP 400") {
		t.Errorf("400 text %s", text)
	}
}

func TestVerifyReadTagsArgumentBounds(t *testing.T) {
	s, _ := neqlServer(t)
	ords := []any{}
	for i := 0; i < 51; i++ {
		ords = append(ords, "station:|slot:/P")
	}
	for _, kv := range [][]any{
		{"ords", ords}, {"ords", []any{}}, {"ord", "slot:/P"},
	} {
		if text, isErr, _ := s.Call(t, "n4_read_tags", kv...); !isErr {
			t.Errorf("%v accepted: %.200s", kv[0], text)
		}
	}
	if text := rpcError(t, s, "n4_read_tags", "ord", "station:|slot:/P", "relations", "some"); !strings.Contains(text,
		"argument relations must be one of direct, all, none; got 'some'") {
		t.Errorf("relations some: %s", text)
	}
}

func TestVerifyDictionariesBudgetAcrossDictionaries(t *testing.T) {
	s, _ := dictServer(t)
	_, _, out := s.Call(t, "n4_list_tag_dictionaries", "max_items", pyjson.IntLiteral("3"))
	got := marshal(t, out)
	tr, _ := out.Get("truncated")
	if tr != true || strings.Contains(got, "tst:cap\", \"info") || !strings.Contains(got, `"name": "Lab"`) {
		t.Errorf("max_items 3: %s", got)
	}
}
