package readtools_test

// n4_alarm_health through the in-memory station: the two reads it sends and the lists it
// groups from them.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	healthWindowOrd = "alarm:|bql:select source, normalTime.encodeToString where timestamp.millis >= 1794852000000|view:file:ITableToCsv"
	healthOpenOrd   = "alarm:|bql:select source, sourceState.encodeToString, ackState.encodeToString, alarmClass, " +
		"timestamp.encodeToString where " + alarmOpen + "|view:file:ITableToCsv"
	nullTime = "1969-12-31T18:00:00.000-06:00"
)

// healthServer answers a window where Pump chatters (6 records, 5 back to normal: 11
// transitions) and Fan alarmed once, and three open records: Fan still offnormal, Valve back to
// normal but unacknowledged since 2026-11-10, Door acked.
func healthServer(t *testing.T) *readtoolstest.Server {
	t.Helper()
	fixedNow(t, "2026-11-17T18:00:00Z")
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	window := "Source,Encode To String\n"
	for i := 0; i < 6; i++ {
		back := "2026-11-17T08:00:00.000-06:00"
		if i == 5 {
			back = nullTime
		}
		window += "local:|station:|slot:/Plant/Pump," + back + "\n"
	}
	window += "local:|station:|slot:/Plant/Fan," + nullTime + "\n"
	open := "Source,Encode To String,Encode To String,Alarm Class,Encode To String\n" +
		"local:|station:|slot:/Plant/Fan,offnormal,unacked,Critical,2026-11-17T07:00:00.000-06:00\n" +
		"local:|station:|slot:/Plant/Valve,normal,unacked,defaultAlarmClass,2026-11-10T07:00:00.000-06:00\n" +
		"local:|station:|slot:/Plant/Door,offnormal,acked,defaultAlarmClass,2026-11-01T07:00:00.000-06:00\n"
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "select source, normalTime", Body: window},
		{Needle: "select source, sourceState", Body: open},
	}
	return s
}

func TestAlarmHealthGroupsTheWindowAndTheOpenRecords(t *testing.T) {
	s := healthServer(t)
	out := callOK(t, s, "n4_alarm_health")
	if got := sent(s); strings.Join(got, "\n") != healthWindowOrd+"\n"+healthOpenOrd {
		t.Errorf("ORDs sent %q", got)
	}
	for k, want := range map[string]string{
		"chattering":    "[{'source': 'station:|slot:/Plant/Pump', 'transitions': 11, 'records': 6}]",
		"top_sources":   "[{'source': 'station:|slot:/Plant/Pump', 'count': 6}, {'source': 'station:|slot:/Plant/Fan', 'count': 1}]",
		"offnormal":     "[{'source': 'station:|slot:/Plant/Door', 'source_state': 'offnormal', 'alarm_class': 'defaultAlarmClass', 'since': '2026-11-01T07:00:00.000-06:00', 'records': 1}, {'source': 'station:|slot:/Plant/Fan', 'source_state': 'offnormal', 'alarm_class': 'Critical', 'since': '2026-11-17T07:00:00.000-06:00', 'records': 1}]",
		"unacked_older": "[{'source': 'station:|slot:/Plant/Valve', 'alarm_class': 'defaultAlarmClass', 'ack_state': 'unacked', 'alarm_time': '2026-11-10T07:00:00.000-06:00', 'age_days': 7}]",
		"counts":        "{'records_in_window': 7, 'sources_in_window': 2, 'chattering': 1, 'open_records': 3, 'offnormal': 2, 'unacked_older': 1, 'unreadable_alarm_times': 0}",
	} {
		if got := pyjson.Repr(field(out, k)); got != want {
			t.Errorf("%s\n got %s\nwant %s", k, got, want)
		}
	}
	if field(out, "truncated") != false || field(out, "window") != "24h" || field(out, "now") != "2026-11-17T18:00:00.000Z" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	obs := s.Ctx.Observations[len(s.Ctx.Observations)-1]
	if field(obs, "tool") != "n4_alarm_health" || field(obs, "rows") != 10 {
		t.Errorf("observation %s", pyjson.Repr(obs))
	}
}

func TestAlarmHealthCutsListsToTopAndAppliesItsArguments(t *testing.T) {
	s := healthServer(t)
	out := callOK(t, s, "n4_alarm_health", "window", "7d", "chatter_threshold", pyjson.IntLiteral("1"),
		"unacked_days", pyjson.IntLiteral("20"), "top", pyjson.IntLiteral("1"))
	if !strings.Contains(sent(s)[0], "timestamp.millis >= 1794333600000|") {
		t.Errorf("7d window: %q", sent(s)[0])
	}
	if len(field(out, "chattering").([]any)) != 1 || len(field(out, "unacked_older").([]any)) != 0 ||
		field(out, "truncated") != true || pyjson.Repr(field(out, "truncated_note")) !=
		"['lists are cut to top=1 entries: chattering (2), offnormal (2), top_sources (2); counts give the full numbers']" {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	if got := pyjson.Repr(field(field(out, "counts").(*pyjson.Object), "chattering")); got != "2" {
		t.Errorf("chattering count %s", got)
	}
}

func TestAlarmHealthRefusesAWindowThatIsNotASpan(t *testing.T) {
	for _, w := range []string{"0h", "2026-11-15T00:00:00Z", "24 h", "1y"} {
		s := healthServer(t)
		if text := callFail(t, s, "n4_alarm_health", "window", w); !strings.Contains(text, "window must be a span back from now") || len(sent(s)) != 0 {
			t.Errorf("%q: %s, sent %q", w, text, sent(s))
		}
	}
	rpcError(t, healthServer(t), "n4_alarm_health", "top", pyjson.IntLiteral("51"))
}
