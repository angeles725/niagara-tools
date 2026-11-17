package readtools_test

// n4_read_history summary=true, retro 2026-11-16 M2 ("a cuanto llego" in one call): count, min,
// max, first, last and per-status counts over the whole range, whatever limit cut the rows.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

const (
	summaryQuery   = "|bql:select timestamp.encodeToString, value.encodeToString, status order by timestamp desc|view:file:ITableToCsv"
	aggregateQuery = "|bql:select min(value), max(value) where status.null = false|view:file:ITableToCsv"
)

// summaryRows is the summary pass of values (oldest first in the argument), answered newest
// first with machine-encoded timestamps and values, one record a minute.
func summaryRows(values []string, statuses []string) string {
	var b strings.Builder
	b.WriteString("Encode To String,Encode To String,Status\n")
	for i := len(values) - 1; i >= 0; i-- {
		fmt.Fprintf(&b, "2026-11-16T02:%02d:00.000-06:00,%s,%s\n", 20+i, values[i], statuses[i])
	}
	return b.String()
}

// summaryServer is historyServer whose station also answers the summary pass and the aggregate.
func summaryServer(t *testing.T, values, statuses []string) *readtoolstest.Server {
	t.Helper()
	s := historyServer(t, len(values))
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{
		{Needle: "value.encodeToString", Body: summaryRows(values, statuses)},
		{Needle: "min(value)", Body: "min(value),max(value)\n\"-1,00\",\"22,25\"\n"},
	}, s.Station.OrdRoutes...)
	return s
}

var (
	peakValues   = []string{"15.0", "22.25", "23.1", "-1.0", "22.25"}
	peakStatuses = []string{"{ok}", "{ok}", "{stale}", "{ok}", "{ok}"}
)

func TestReadHistorySummarizesTheWholeRangeWhateverTheLimit(t *testing.T) {
	s := summaryServer(t, peakValues, peakStatuses)
	out := readOK(t, s, "history_id", "/FakeStation/NW", "limit", pyjson.IntLiteral("2"), "summary", true)
	want := []string{"history:/FakeStation/NW" + recordsQuery, "history:/FakeStation/NW" + countQuery,
		"history:/FakeStation/NW" + summaryQuery}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("ORDs sent %q, want %q", got, want)
	}
	if field(out, "row_count") != 2 || field(out, "truncated") != true {
		t.Errorf("rows %s", pyjson.Repr(out))
	}
	got := pyjson.Repr(field(out, "summary"))
	wantSummary := "{'count': 5, 'complete': True, 'records_scanned': 5, 'numeric': True, " +
		"'min': {'value': -1.0, 'timestamp': '2026-11-16T02:23:00.000-06:00'}, " +
		"'max': {'value': 23.1, 'timestamp': '2026-11-16T02:22:00.000-06:00'}, " +
		"'first': {'value': 15.0, 'timestamp': '2026-11-16T02:20:00.000-06:00', 'status': '{ok}'}, " +
		"'last': {'value': 22.25, 'timestamp': '2026-11-16T02:24:00.000-06:00', 'status': '{ok}'}, " +
		"'status_counts': {'{ok}': 4, '{stale}': 1}}"
	if got != wantSummary {
		t.Errorf("summary %s\nwant    %s", got, wantSummary)
	}
}

func TestReadHistorySummaryTakesTheEarliestRecordOfATie(t *testing.T) {
	out := readOK(t, summaryServer(t, []string{"5.0", "9.0", "5.0", "9.0"}, []string{"{ok}", "{ok}", "{ok}", "{ok}"}),
		"history_id", "/FakeStation/NW", "summary", true)
	got := pyjson.Repr(field(out, "summary"))
	for _, part := range []string{"'min': {'value': 5.0, 'timestamp': '2026-11-16T02:20:00.000-06:00'}",
		"'max': {'value': 9.0, 'timestamp': '2026-11-16T02:21:00.000-06:00'}"} {
		if !strings.Contains(got, part) {
			t.Errorf("summary %s lacks %s", got, part)
		}
	}
}

func TestReadHistorySummaryOverMoreRecordsThanItScansSaysSo(t *testing.T) {
	s := summaryServer(t, peakValues, peakStatuses)
	out := readOK(t, s, "history_id", "/FakeStation/NW", "summary", true, "summary_max_rows", pyjson.IntLiteral("3"))
	sum := field(out, "summary").(*pyjson.Object)
	if field(sum, "complete") != false || field(sum, "records_scanned") != 3 || field(sum, "count") != 5 {
		t.Fatalf("summary %s", pyjson.Repr(sum))
	}
	if field(sum, "range_min") != "-1,00" || field(sum, "range_max") != "22,25" {
		t.Errorf("whole-range aggregate %s", pyjson.Repr(sum))
	}
	note, _ := field(sum, "note").(string)
	if !strings.Contains(note, "newest 3 of 5 records") || !strings.Contains(note, "select min(value), max(value)") {
		t.Errorf("note %q", note)
	}
	if last := sent(s)[len(sent(s))-1]; last != "history:/FakeStation/NW"+aggregateQuery {
		t.Errorf("last ORD %q", last)
	}
}

func TestReadHistorySummaryOfNonNumericOrEmptyHistories(t *testing.T) {
	out := readOK(t, summaryServer(t, []string{"true", "false", "true"}, []string{"{ok}", "{ok}", "{ok}"}),
		"history_id", "/FakeStation/NW", "summary", true)
	sum := field(out, "summary").(*pyjson.Object)
	note, _ := field(sum, "note").(string)
	if field(sum, "numeric") != false || field(sum, "min") != nil || field(sum, "max") != nil ||
		!strings.Contains(note, "not numeric") || !strings.Contains(note, "'true'") {
		t.Errorf("non-numeric summary %s", pyjson.Repr(sum))
	}
	if first := pyjson.Repr(field(sum, "first")); first != "{'value': 'true', 'timestamp': '2026-11-16T02:20:00.000-06:00', 'status': '{ok}'}" {
		t.Errorf("first %s", first)
	}
	s := historyServer(t, 0)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "value.encodeToString",
		Body: "timestamp.encodeToString,value.encodeToString,status\n"}}, s.Station.OrdRoutes...)
	sum = field(readOK(t, s, "history_id", "/FakeStation/NW", "summary", true), "summary").(*pyjson.Object)
	if pyjson.Repr(sum) != "{'count': 0, 'complete': True, 'records_scanned': 0, 'numeric': False, 'min': None, "+
		"'max': None, 'first': None, 'last': None, 'status_counts': {}, 'note': 'the range holds no records'}" {
		t.Errorf("empty summary %s", pyjson.Repr(sum))
	}
}

func TestReadHistoryTakesARelativeStartAndABareName(t *testing.T) {
	fixedNow(t, "2026-11-16T12:00:00Z")
	s := historyServer(t, 1)
	out := readOK(t, s, "history_id", "NW", "start", "24h")
	if got := sent(s)[0]; got != "history:/FakeStation/NW?start=2026-11-15T12:00:00.000Z"+recordsQuery {
		t.Errorf("ORD sent %q", got)
	}
	if field(out, "history_id") != "/FakeStation/NW" || field(out, "range") != "start=2026-11-15T12:00:00.000Z" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	for _, kv := range []any{"5x", "0d"} {
		s := historyServer(t, 1)
		if text := readFail(t, s, "history_id", "NW", "start", kv); !strings.Contains(text, "BAbsTime") || len(s.Station.OrdRequests) != 0 {
			t.Errorf("start %v: %s", kv, text)
		}
	}
}

func TestReadHistorySummaryOfOnlyNonFiniteRecordsHasNoMinMax(t *testing.T) {
	out := readOK(t, summaryServer(t, []string{"nan", "inf"}, []string{"{fault}", "{fault}"}),
		"history_id", "/FakeStation/NW", "summary", true)
	sum := field(out, "summary").(*pyjson.Object)
	if field(sum, "min") != nil || field(sum, "max") != nil || field(sum, "count") != 2 {
		t.Errorf("summary %s", pyjson.Repr(sum))
	}
}
