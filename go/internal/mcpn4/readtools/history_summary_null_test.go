package readtools_test

// Retro 2026-11-16 M16: a {null}-status record holds no value (BStatusValue renders it "-"), so
// the summary counts it but leaves it out of min, max, first and last, and says so.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// A new COV history starts with two 0.0 {null} records.
var (
	nullStartValues   = []string{"0.0", "0.0", "10.0", "25.0", "30.0", "12.0"}
	nullStartStatuses = []string{"{null}", "{null}", "{ok}", "{alarm}", "{alarm}", "{ok}"}
)

const nullStartSummary = "{'count': 6, 'complete': True, 'records_scanned': 6, 'numeric': True, " +
	"'min': {'value': 10.0, 'timestamp': '2026-11-16T02:22:00.000-06:00'}, " +
	"'max': {'value': 30.0, 'timestamp': '2026-11-16T02:24:00.000-06:00'}, " +
	"'first': {'value': 10.0, 'timestamp': '2026-11-16T02:22:00.000-06:00', 'status': '{ok}'}, " +
	"'last': {'value': 12.0, 'timestamp': '2026-11-16T02:25:00.000-06:00', 'status': '{ok}'}, " +
	"'status_counts': {'{ok}': 2, '{alarm}': 2, '{null}': 2}, " +
	"'excluded_from_extremes': {'null': 2, 'nan_inf': 0}, " +
	"'first_record': {'value': 0.0, 'timestamp': '2026-11-16T02:20:00.000-06:00', 'status': '{null}'}, " +
	"'note': '2 record(s) with status null (no value) and 0 nan/inf left out of min, max, first and last; " +
	"first_record and last_record are the raw ends when they differ'}"

func TestReadHistorySummaryLeavesNullStatusRecordsOutOfMinMax(t *testing.T) {
	s := summaryServer(t, nullStartValues, nullStartStatuses)
	out := readOK(t, s, "history_id", "/FakeStation/NW", "summary", true)
	if got := pyjson.Repr(field(out, "summary")); got != nullStartSummary {
		t.Errorf("summary %s\nwant    %s", got, nullStartSummary)
	}
}

// A null flag among others ({alarm,null}, quoted in the CSV) is still null; a status with facets
// keeps its flags.
func TestReadHistorySummaryReadsTheNullFlagAmongOthers(t *testing.T) {
	s := summaryServer(t, []string{"0.0", "7.5", "9.0"}, []string{`"{alarm,null}"`, "{ok}", "{null} @ def"})
	sum := field(readOK(t, s, "history_id", "/FakeStation/NW", "summary", true), "summary").(*pyjson.Object)
	got := pyjson.Repr(sum)
	for _, part := range []string{"'min': {'value': 7.5", "'max': {'value': 7.5",
		"'excluded_from_extremes': {'null': 2, 'nan_inf': 0}", "'last_record': {'value': 9.0"} {
		if !strings.Contains(got, part) {
			t.Errorf("summary %s lacks %s", got, part)
		}
	}
}

// Every record null: min, max, first and last are null, the raw ends stay.
func TestReadHistorySummaryOfOnlyNullRecordsHasNoExtremes(t *testing.T) {
	s := summaryServer(t, []string{"0.0", "0.0"}, []string{"{null}", "{null}"})
	sum := field(readOK(t, s, "history_id", "/FakeStation/NW", "summary", true), "summary").(*pyjson.Object)
	if field(sum, "min") != nil || field(sum, "max") != nil || field(sum, "first") != nil || field(sum, "last") != nil ||
		field(sum, "first_record") == nil || field(sum, "last_record") == nil || field(sum, "count") != 2 {
		t.Errorf("summary %s", pyjson.Repr(sum))
	}
}

// n4_event_report summarizes its window through the same summary.
func TestEventReportLeavesNullStatusRecordsOutOfMinMax(t *testing.T) {
	s := eventServer(t, eventExts)
	s.Station.OrdRoutes[3].Body = summaryRows(nullStartValues, nullStartStatuses)
	out := callOK(t, s, "n4_event_report", "source", "station:|slot:/Plant/Comp1/AmpsAlarm")
	if got := pyjson.Repr(field(events(t, out)[0], "summary")); got != nullStartSummary {
		t.Errorf("event summary %s\nwant          %s", got, nullStartSummary)
	}
}
