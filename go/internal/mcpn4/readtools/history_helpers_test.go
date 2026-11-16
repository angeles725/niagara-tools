package readtools_test

// Helpers of the n4_read_history tests: the station's recorded answers and the calls.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	recordsQuery = "|bql:select timestamp, value, status order by timestamp desc|view:file:ITableToCsv"
	countQuery   = "|bql:select count(*)|view:file:ITableToCsv"
	extQuery     = "|bql:select slotPath, historyConfig.id from history:HistoryExt|view:file:ITableToCsv"
)

// records is an ITableToCsv answer of n records, newest first, as a 4.15 station renders them.
func records(n int) string {
	var b strings.Builder
	b.WriteString("Timestamp,Value,Status\n")
	for i := n; i > 0; i-- {
		fmt.Fprintf(&b, "15 nov 2026 20:%02d:35 CST,\"%d,0\",{ok}\n", i, i)
	}
	return b.String()
}

// historyServer is a connected server whose station answers the records, the count and the
// extension lookup.
func historyServer(t *testing.T, n int) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "order by timestamp desc", Body: records(n)},
		{Needle: "select count(*)", Body: fmt.Sprintf("count(toString)\n%d\n", n)},
		{Needle: "history:HistoryExt", Body: "Slot Path,History Config Id\n" +
			"slot:/Folder/NW/NumericInterval,/FakeStation/NW\nslot:/Folder/NW/Other,/FakeStation/Other\n"},
	}
	return s
}

func readOK(t *testing.T, s *readtoolstest.Server, kv ...any) *pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, "n4_read_history", kv...)
	if isErr {
		t.Fatalf("n4_read_history: %s", text)
	}
	return out
}

func readFail(t *testing.T, s *readtoolstest.Server, kv ...any) string {
	t.Helper()
	text, isErr, _ := s.Call(t, "n4_read_history", kv...)
	if !isErr {
		t.Fatalf("n4_read_history succeeded: %s", text)
	}
	return text
}

func field(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }

func sent(s *readtoolstest.Server) []string {
	var out []string
	for _, r := range s.Station.OrdRequests {
		out = append(out, r.Ord)
	}
	return out
}
