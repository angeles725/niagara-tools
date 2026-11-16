package readtools_test

// Every value n4_read_alarms and n4_read_history compose into an ORD is checked as the station
// reads it: $xx / $uxxxx decoded first (the /ord/ path is decoded before the BQL is parsed), then
// refused with no request sent when it decodes to a quote, a space, '|' or a control character.
// Live, 4.15.6: source=station:|slot:/nomatch$27$20or$20source$20like$20$27$2a matched every alarm.

import (
	"strings"
	"testing"
)

func TestReadAlarmsRefusesAnEscapedQuoteInSourceAndAlarmClass(t *testing.T) {
	for _, c := range []struct{ arg, value string }{
		{"source", "station:|slot:/nomatch$27$20or$20source$20like$20$27$2a"},
		{"source", "station:|slot:/Drivers/A$7cbql"},
		{"source", "station:|slot:/Drivers/A$u0027"},
		{"source", "station:|slot:/Drivers/A$2427"}, // an escaped escape: $24 is '$', then 27
		{"alarm_class", "x$27$20or$20alarmClass$20$21$3d$20$27"},
		{"alarm_class", "x$2427"},
		{"alarm_class", "x$0a"},
		{"source_glob", "*$2427*"},
	} {
		s := alarmServer(t, 3)
		msg := callFail(t, s, "n4_read_alarms", c.arg, c.value)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(msg, c.arg) || !strings.Contains(msg, c.value) {
			t.Errorf("%s %q: %d requests, message %s", c.arg, c.value, len(s.Station.OrdRequests), msg)
		}
	}
	s := alarmServer(t, 3) // a harmless escape ('-' and 'A') still reads
	callOK(t, s, "n4_read_alarms", "source", "station:|slot:/Drivers/Net$2dA", "alarm_class", "class$41")
	if len(s.Station.OrdRequests) != 2 {
		t.Errorf("harmless escapes: %d requests", len(s.Station.OrdRequests))
	}
}

func TestReadHistoryRefusesAnEscapedOrdSeparatorAndNamesTheCallersText(t *testing.T) {
	for _, id := range []string{"/FakeStation/NW$7cbql:select$20x", "NW$7cbql", "NW$247c", "NW$zz"} {
		s := historyServer(t, 1)
		msg := readFail(t, s, "history_id", id)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(msg, "history_id") || !strings.Contains(msg, id) {
			t.Errorf("history_id %q: %d requests, message %s", id, len(s.Station.OrdRequests), msg)
		}
	}
	s := historyServer(t, 1)
	msg := readFail(t, s, "history_id", "N W")
	if !strings.Contains(msg, "'N W'") || strings.Contains(msg, "/FakeStation/") || len(s.Station.OrdRequests) != 0 {
		t.Errorf("bare name refusal %s", msg)
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
