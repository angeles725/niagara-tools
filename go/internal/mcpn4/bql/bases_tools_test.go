package bql_test

// n4_bql_query over the alarm: and history: bases (retro 2026-11-16 M1): one GET of the same
// ITableToCsv view, rows in the station-base reply shape; a non-select or an unknown base sends
// nothing.

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

const alarmCSV = "Timestamp,Source,Source State\n16 nov 2026 00:08:56 CST,local:|station:|slot:/Lab/P/AlarmExt,Offnormal\n"

func TestQueryReadsTheAlarmBaseAsRows(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "alarm:|bql:", Body: alarmCSV}}
	out := ok(t, s, "n4_bql_query", "base", "alarm:", "query", "select timestamp, source, sourceState")
	if repr(get(out, "columns")) != "['Timestamp', 'Source', 'Source State']" || get(out, "row_count") != 1 ||
		get(out, "base") != "alarm:" || get(out, "truncated") != false {
		t.Errorf("reply %s", repr(out))
	}
	if got := s.Station.OrdRequests[0].Ord; got != "alarm:|bql:select timestamp, source, sourceState|view:file:ITableToCsv" {
		t.Errorf("ORD sent %q", got)
	}
}

func TestQueryReadsAHistoryBaseAsRows(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "history:/FakeStation/NW|bql:", Body: "Timestamp,Value\n16 nov 2026 00:00:00 CST,21.5\n"}}
	out := ok(t, s, "n4_bql_query", "base", "history:/FakeStation/NW", "query", "select timestamp, value where timestamp.millis >= 0")
	if repr(get(out, "rows")) != "[{'Timestamp': '16 nov 2026 00:00:00 CST', 'Value': '21.5'}]" {
		t.Errorf("reply %s", repr(out))
	}
	if got := s.Station.OrdRequests[0].Ord; got != "history:/FakeStation/NW|bql:select timestamp, value where timestamp.millis >= 0|view:file:ITableToCsv" {
		t.Errorf("ORD sent %q", got)
	}
}

func TestATableBaseRefusesANonSelectAndAnUnknownBaseSendingNothing(t *testing.T) {
	s := bqlServer(t, "")
	has(t, fail(t, s, "n4_bql_query", "base", "alarm:", "query", "delete *"), "query", "select")
	has(t, fail(t, s, "n4_bql_query", "base", "file:^px", "query", "select *"), "base", "alarm:", "history:")
	has(t, fail(t, s, "n4_bql_query", "base", "history:/S", "query", "select *"), "base", "history")
	if len(s.Station.OrdRequests) != 0 {
		t.Errorf("requests sent: %+v", s.Station.OrdRequests)
	}
}

func TestInventoryStillRefusesAnAlarmBase(t *testing.T) {
	s := bqlServer(t, "")
	fail(t, s, "n4_inventory", "base", "alarm:")
	if len(s.Station.OrdRequests) != 0 {
		t.Errorf("requests sent: %+v", s.Station.OrdRequests)
	}
}
