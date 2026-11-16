package readtools_test

// n4_event_report, retro 2026-11-16 P3 ("a cuanto llego" took 6 steps): per alarm event of a
// source, its alarm database times and its history's summary over the event window, one call.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestEventReportJoinsEachAlarmEventToItsHistoryPeak(t *testing.T) {
	s := eventServer(t, eventExts)
	out := callOK(t, s, "n4_event_report", "source", "station:|slot:/Plant/Comp1/AmpsAlarm")
	want := []string{eventSelect + " where " + eventSource + alarmOrder,
		"alarm:|bql:select count(*) where " + eventSource + "|view:file:ITableToCsv",
		eventExtOrd, windowActive, windowClosed}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("ORDs sent\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if field(out, "event_count") != 2 || field(out, "total_events") != 2 || field(out, "truncated") != false ||
		field(out, "margin_minutes") != 15 {
		t.Errorf("report %s", pyjson.Repr(out))
	}
	evs := events(t, out)
	active := pyjson.Repr(evs[0])
	for _, part := range []string{"'uuid': 'aa000000-0000-0000-0000-000000000002'",
		"'alarm_time': '2026-11-16T10:00:00.000-06:00', 'normal_time': None, 'ack_time': None, 'acked_by': None",
		"'window': {'start': '2026-11-16T09:45:00.000-06:00', 'end': '2026-11-16T11:00:00.000-06:00', 'end_is_now': True}",
		"'history': {'id': '/FakeStation/Comp1_Amps', 'found_by': 'history_ext', 'ext_slot': 'slot:/Plant/Comp1/NumericCov'}",
		"'max': {'value': 41.2, 'timestamp': '2026-11-16T10:20:00.000-06:00'}"} {
		if !strings.Contains(active, part) {
			t.Errorf("active event %s\nlacks %s", active, part)
		}
	}
	closed := pyjson.Repr(evs[1])
	for _, part := range []string{"'source_state': 'normal', 'ack_state': 'acked'",
		"'alarm_time': '2026-11-16T08:00:00.000-06:00', 'normal_time': '2026-11-16T08:30:00.000-06:00', " +
			"'ack_time': '2026-11-16T08:40:00.000-06:00', 'acked_by': 'admin', 'records': 1",
		"'window': {'start': '2026-11-16T07:45:00.000-06:00', 'end': '2026-11-16T08:45:00.000-06:00', 'end_is_now': False}",
		"'count': 3", "'min': {'value': 10.0, 'timestamp': '2026-11-16T07:50:00.000-06:00'}",
		"'max': {'value': 38.75, 'timestamp': '2026-11-16T08:10:00.000-06:00'}"} {
		if !strings.Contains(closed, part) {
			t.Errorf("closed event %s\nlacks %s", closed, part)
		}
	}
}

func TestEventReportSaysWhenTheSourceHasNoHistory(t *testing.T) {
	s := eventServer(t, "Slot Path,History Config Id\n")
	out := callOK(t, s, "n4_event_report", "source", "station:|slot:/Plant/Comp1/AmpsAlarm")
	for _, ev := range events(t, out) {
		if field(ev, "history") != nil || field(ev, "summary") != nil ||
			!strings.Contains(pyjson.Str(field(ev, "summary_note")), "no history extension") {
			t.Errorf("event %s", pyjson.Repr(ev))
		}
	}
	if n := strings.Count(strings.Join(sent(s), "\n"), "history:HistoryExt"); n != 1 {
		t.Errorf("the extension lookup ran %d times, want once per source", n)
	}
}

func TestEventReportReadsAnExplicitHistoryWithAMargin(t *testing.T) {
	s := eventServer(t, eventExts)
	s.Station.OrdRoutes[3].Needle = "start=2026-11-16T15:55" // margin 5 minutes
	s.Station.OrdRoutes[4].Needle = "start=2026-11-16T13:55"
	out := callOK(t, s, "n4_event_report", "source_glob", "*slot:/Plant/Comp1/*", "history_id", "Comp1_Amps",
		"margin_minutes", pyjson.IntLiteral("5"), "max_events", pyjson.IntLiteral("2"))
	if got := strings.Join(sent(s), "\n"); strings.Contains(got, "HistoryExt") ||
		!strings.Contains(got, "end=2026-11-16T14:35:00.000Z") {
		t.Errorf("ORDs sent %s", got)
	}
	ev := pyjson.Repr(events(t, out)[1])
	if !strings.Contains(ev, "'history': {'id': '/FakeStation/Comp1_Amps', 'found_by': 'history_id'}") ||
		!strings.Contains(ev, "'value': 38.75") {
		t.Errorf("event %s", ev)
	}
}

func TestEventReportKeepsTheOtherEventsWhenOneWindowFails(t *testing.T) {
	s := eventServer(t, eventExts)
	s.Station.OrdRoutes[3].Code = 400
	out := callOK(t, s, "n4_event_report", "source", "station:|slot:/Plant/Comp1/AmpsAlarm")
	evs := events(t, out)
	if field(evs[0], "summary") != nil || !strings.Contains(pyjson.Str(field(evs[0], "summary_note")), "HTTP 400") {
		t.Errorf("failed window %s", pyjson.Repr(evs[0]))
	}
	if field(evs[1], "summary") == nil {
		t.Errorf("the other event lost its summary: %s", pyjson.Repr(evs[1]))
	}
}

func TestEventReportRefusesBadArgumentsBeforeAnyRequest(t *testing.T) {
	s := eventServer(t, eventExts)
	for name, kv := range map[string][]any{
		"no source":     {},
		"both sources":  {"source", "station:|slot:/Plant", "source_glob", "*slot:/Plant/*"},
		"escaped quote": {"source", "station:|slot:/Plant/a$27b"},
		"glob with $":   {"source_glob", "*slot:/Plant/$27*"},
		"bad history":   {"source", "station:|slot:/Plant", "history_id", "a|b"},
		"since > until": {"source", "station:|slot:/Plant", "since", "1h", "until", "2h"},
	} {
		callFail(t, s, "n4_event_report", kv...)
		if len(s.Station.OrdRequests) != 0 {
			t.Errorf("%s: %d request(s) reached the station", name, len(s.Station.OrdRequests))
		}
	}
}
