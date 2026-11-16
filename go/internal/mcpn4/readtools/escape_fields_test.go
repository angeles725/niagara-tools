package readtools_test

// Niagara decodes $xx / $uxxxx only inside a BQL body (BqlQuery.make), so every value that goes
// into BQL is checked on its decoded form and refused with no request sent; ORD paths and history
// ids are matched escaped and are never decoded. Live, 4.15.6:
// alarm_class=x$27$20or$20alarmClass$20like$20$27$2a matched 27 of 27 records, and
// select uuid$20from$20control:NumericPoint on alarm: passed the no-from rule.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

func TestReadAlarmsRefusesAFilterThatBqlDecodesToAQuote(t *testing.T) {
	for _, c := range []struct{ arg, value string }{
		{"source", "station:|slot:/nomatch$27$20or$20source$20like$20$27$2a"},
		{"source", "station:|slot:/Drivers/A$2a"},
		{"source", "station:|slot:/Drivers/A$u0027"},
		{"source", "station:|slot:/Drivers/A$20B"},
		{"alarm_class", "x$27$20or$20alarmClass$20like$20$27$2a"},
		{"alarm_class", "x$0a"},
		{"alarm_class", "x$u0022"},
	} {
		s := alarmServer(t, 3)
		msg := callFail(t, s, "n4_read_alarms", c.arg, c.value)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(msg, c.arg) || !strings.Contains(msg, c.value) {
			t.Errorf("%s %q: %d requests, message %s", c.arg, c.value, len(s.Station.OrdRequests), msg)
		}
	}
}

func TestReadAlarmsSendsAnEscapedEscapeAndAHarmlessEscapeAsGiven(t *testing.T) {
	// $2427 decodes once to the literal text $27 (no quote), as BqlQuery reads it; $2d is '-'.
	s := alarmServer(t, 3)
	callOK(t, s, "n4_read_alarms", "source", "station:|slot:/Drivers/A$2427", "alarm_class", "x$2427$2d")
	if got := sent(s)[0]; !strings.Contains(got, "source like 'slot:/Drivers/A$2427'") || !strings.Contains(got, "alarmClass = 'x$2427$2d'") {
		t.Errorf("ORD sent %q", got)
	}
}

func TestBqlQueryChecksTheDecodedQueryText(t *testing.T) {
	for _, c := range []struct{ base, query, want string }{
		{"alarm:", "select uuid$20from$20control:NumericPoint", "from clause"},
		{"alarm:", "select uuid$u0020from$20x", "from clause"},
		{"station:|slot:/Drivers", "select slotPath from control:NumericPoint where name = 'a$7cb'", "'|'"},
		{"station:|slot:/Drivers", "select slotPath from control:NumericPoint where name = 'a$0ab'", "control characters"},
		{"history:/FakeStation/NW", "select timestamp$zz", "malformed $ escape"},
	} {
		s := alarmServer(t, 3)
		msg := callFail(t, s, "n4_bql_query", "base", c.base, "query", c.query)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(msg, c.want) {
			t.Errorf("%s %q: %d requests, message %s", c.base, c.query, len(s.Station.OrdRequests), msg)
		}
	}
}

func TestNeqlSelectChecksTheDecodedText(t *testing.T) {
	s, st := neqlServer(t, readtoolstest.OrdRoute{Needle: "neql:", Code: 400})
	for _, bql := range []string{"select a$20from$20b", "select a where x = 'a$7cb'", "select a$zz"} {
		if text, isErr, _ := s.Call(t, "n4_neql_query", "neql", "n:point", "bql", bql); !isErr {
			t.Errorf("bql %q accepted: %s", bql, text)
		}
	}
	if len(st.OrdRequests) != 0 {
		t.Errorf("a refused select reached the station: %v", st.OrdRequests)
	}
}

func TestReadHistoryNeverDecodesTheIdAndNamesTheCallersText(t *testing.T) {
	// A history id is matched escaped (SlotPath), so $7c there is a name character, not '|'.
	s := historyServer(t, 1)
	readOK(t, s, "history_id", "NW$7cx")
	if got := sent(s)[0]; !strings.HasPrefix(got, "history:/FakeStation/NW$7cx|bql:") {
		t.Errorf("ORD sent %q", got)
	}
	for _, id := range []string{"N W", "NW$zz"} {
		s := historyServer(t, 1)
		msg := readFail(t, s, "history_id", id)
		if !strings.Contains(msg, "'"+id+"'") || strings.Contains(msg, "/FakeStation/") || len(s.Station.OrdRequests) != 0 {
			t.Errorf("bare name %q refusal %s", id, msg)
		}
	}
}

func TestReadHistoryRefusesAStartLaterThanEndAndEchoesNoRangeForAPeriod(t *testing.T) {
	s := historyServer(t, 1)
	msg := readFail(t, s, "history_id", "/FakeStation/NW", "start", "2026-11-16T00:00:00Z", "end", "2026-11-15T00:00:00Z")
	if !strings.Contains(msg, "start must not be later than end") || len(s.Station.OrdRequests) != 0 {
		t.Errorf("start after end: %s", msg)
	}
	out := readOK(t, historyServer(t, 1), "history_id", "/FakeStation/NW", "period", "today")
	if _, ok := out.Get("range"); ok {
		t.Errorf("range echoed beside period: %v", field(out, "range"))
	}
}
