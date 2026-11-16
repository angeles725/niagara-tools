package readtools_test

// n4_read_history through the in-memory station: the ORDs it sends, the rows it returns and
// its refusals.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

func TestReadHistoryReturnsTheLastRecordsNewestFirst(t *testing.T) {
	s := historyServer(t, 5)
	out := readOK(t, s, "history_id", "/FakeStation/NW", "limit", pyjson.IntLiteral("2"))
	want := []string{"history:/FakeStation/NW" + recordsQuery, "history:/FakeStation/NW" + countQuery}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("ORDs sent %q, want %q", got, want)
	}
	rows := field(out, "rows").([]any)
	if len(rows) != 2 || field(out, "row_count") != 2 || field(out, "truncated") != true {
		t.Fatalf("reply %s", pyjson.Repr(out))
	}
	first := rows[0].(*pyjson.Object)
	if pyjson.Repr(first) != "{'timestamp': '15 nov 2026 20:05:35 CST', 'value': '5,0', 'status': '{ok}'}" {
		t.Errorf("newest row %s", pyjson.Repr(first))
	}
	if field(out, "order") != "newest_first" || field(out, "history_id") != "/FakeStation/NW" ||
		pyjson.Repr(field(out, "columns")) != "['Timestamp', 'Value', 'Status']" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	if field(out, "total_records") != 5 {
		t.Errorf("total_records %v", field(out, "total_records"))
	}
	obs := s.Ctx.Observations[len(s.Ctx.Observations)-1]
	if field(obs, "tool") != "n4_read_history" || field(obs, "ord") != "/FakeStation/NW" || field(obs, "rows") != 2 {
		t.Errorf("observation %s", pyjson.Repr(obs))
	}
}

func TestReadHistoryDefaultsToTwentyRecordsAndKeepsAShortHistoryWhole(t *testing.T) {
	out := readOK(t, historyServer(t, 25), "history_id", "/FakeStation/NW")
	if field(out, "row_count") != 20 || field(out, "truncated") != true {
		t.Errorf("25 records: %v rows, truncated %v", field(out, "row_count"), field(out, "truncated"))
	}
	out = readOK(t, historyServer(t, 3), "history_id", "/FakeStation/NW", "limit", pyjson.IntLiteral("1000"))
	if field(out, "row_count") != 3 || field(out, "truncated") != false {
		t.Errorf("3 records: %v rows, truncated %v", field(out, "row_count"), field(out, "truncated"))
	}
	args, params, msg := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	args.Set("history_id", "/FakeStation/NW")
	args.Set("limit", pyjson.IntLiteral("1001"))
	params.Set("name", "n4_read_history")
	params.Set("arguments", args)
	msg.Set("id", 1)
	msg.Set("method", "tools/call")
	msg.Set("params", params)
	if reply := pyjson.Repr(historyServer(t, 3).Dispatch(msg)); !strings.Contains(reply, "argument limit must be <= 1000") {
		t.Errorf("limit 1001: %s", reply)
	}
}

func TestReadHistorySendsThePeriodOrTheTimeRange(t *testing.T) {
	s := historyServer(t, 1)
	out := readOK(t, s, "history_id", "/FakeStation/NW", "period", "today")
	want := []string{"history:/FakeStation/NW?period=today" + recordsQuery, "history:/FakeStation/NW?period=today" + countQuery}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") || field(out, "period") != "today" {
		t.Errorf("ORDs sent %q, period %v", got, field(out, "period"))
	}
	s = historyServer(t, 1)
	readOK(t, s, "history_id", "/FakeStation/NW", "start", "2026-11-15T00:00:00.000-06:00", "end", "2026-11-15T12:00:00Z")
	if got := sent(s)[0]; got != "history:/FakeStation/NW?start=2026-11-15T00:00:00.000-06:00;end=2026-11-15T12:00:00Z"+recordsQuery {
		t.Errorf("ORD sent %q", got)
	}
	for _, kv := range [][]any{{"period", "Hoy"}, {"period", "today", "start", "2026-11-15T00:00:00Z"}, {"end", "yesterday"}} {
		s = historyServer(t, 1)
		text := readFail(t, s, append([]any{"history_id", "/FakeStation/NW"}, kv...)...)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(text, "today, last24Hours") && !strings.Contains(text, "BAbsTime") && !strings.Contains(text, "either period") {
			t.Errorf("%v: %s (%d requests)", kv, text, len(s.Station.OrdRequests))
		}
	}
}

func TestReadHistoryResolvesAnExtensionOrd(t *testing.T) {
	s := historyServer(t, 2)
	out := readOK(t, s, "ext_ord", "station:|slot:/Folder/NW/NumericInterval")
	want := []string{"station:|slot:/Folder/NW" + extQuery, "history:/FakeStation/NW" + recordsQuery, "history:/FakeStation/NW" + countQuery}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("ORDs sent %q, want %q", got, want)
	}
	if field(out, "history_id") != "/FakeStation/NW" || field(out, "ext_ord") != "station:|slot:/Folder/NW/NumericInterval" || field(out, "row_count") != 2 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	s = historyServer(t, 2)
	text := readFail(t, s, "ext_ord", "station:|slot:/Folder/NW/NotAnExt")
	if !strings.Contains(text, "station:|slot:/Folder/NW/NotAnExt") || !strings.Contains(text, "history:HistoryExt") || len(s.Station.OrdRequests) != 1 {
		t.Errorf("not an extension: %s (%d requests)", text, len(s.Station.OrdRequests))
	}
	text = readFail(t, historyServer(t, 2), "ext_ord", "history:/FakeStation/NW")
	if !strings.Contains(text, "base must be a plain slot path") {
		t.Errorf("history ORD as ext_ord: %s", text)
	}
}

func TestReadHistoryRefusesABadIdBeforeAnyRequest(t *testing.T) {
	for _, kv := range [][]any{{}, {"history_id", "/FakeStation/NW", "ext_ord", "station:|slot:/A/B"},
		{"history_id", "/FakeStation/NW|bql:select x"}, {"history_id", "FakeStation/NW"}, {"history_id", "/FakeStation/NW?period=today"}} {
		s := historyServer(t, 1)
		text := readFail(t, s, kv...)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(text, "history_id") {
			t.Errorf("%v: %s (%d requests)", kv, text, len(s.Station.OrdRequests))
		}
	}
}

func TestReadHistoryNamesAMissingHistory(t *testing.T) {
	s := historyServer(t, 1)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "/FakeStation/Ghost|", Code: 404}}, s.Station.OrdRoutes...)
	text := readFail(t, s, "history_id", "/FakeStation/Ghost")
	for _, part := range []string{"/FakeStation/Ghost", "HTTP 404", "select historyConfig.id from history:HistoryExt", "n4_bql_query"} {
		if !strings.Contains(text, part) {
			t.Errorf("%q lacks %q", text, part)
		}
	}
	s = historyServer(t, 1)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "/FakeStation/NW|", Code: 401}}, s.Station.OrdRoutes...)
	if text := readFail(t, s, "history_id", "/FakeStation/NW"); strings.Contains(text, "n4_bql_query") {
		t.Errorf("an auth error was reported as a missing history: %s", text)
	}
}

func TestReadHistorySaysWhenTheCountFailed(t *testing.T) {
	s := historyServer(t, 2)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "select count(*)", Code: 500}}, s.Station.OrdRoutes...)
	out := readOK(t, s, "history_id", "/FakeStation/NW")
	note, _ := field(out, "total_records_note").(string)
	if v, ok := out.Get("total_records"); !ok || v != nil || !strings.Contains(note, "HTTP 500") || field(out, "row_count") != 2 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
}
