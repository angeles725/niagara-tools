package readtools_test

// n4_read_alarms through the in-memory station: the ORDs it sends, the rows it returns and its
// refusals.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

func TestReadAlarmsDefaultsToOpenAlarmsNewestFirst(t *testing.T) {
	s := alarmServer(t, 60)
	out := callOK(t, s, "n4_read_alarms")
	want := []string{alarmSelect + " where " + alarmOpen + alarmOrder,
		alarmCount + " where " + alarmOpen + "|view:file:ITableToCsv"}
	if got := sent(s); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("ORDs sent %q, want %q", got, want)
	}
	rows := field(out, "rows").([]any)
	if len(rows) != 50 || field(out, "row_count") != 50 || field(out, "truncated") != true || field(out, "total") != 60 {
		t.Fatalf("reply %s", pyjson.Repr(out))
	}
	first := pyjson.Repr(rows[0].(*pyjson.Object))
	if first != "{'timestamp': '15 nov 2026 18:60:58 CST', 'uuid': 'aa110470-0000-0000-0000-000000000060', "+
		"'source': 'local:|station:|slot:/Drivers/Net/D60', 'source_state': 'Offnormal', 'ack_state': 'Unacked', "+
		"'alarm_class': 'defaultAlarmClass', 'priority': '255', 'msg_text': 'Ping Failed, retry', "+
		"'timestamp_iso': '2026-11-15T18:60:58.000-06:00', 'normal_time': None, 'ack_time': '2026-11-15T19:60:00.000-06:00'}" {
		t.Errorf("newest row %s", first)
	}
	if field(out, "order") != "newest_first" || field(out, "where") != alarmOpen {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	obs := s.Ctx.Observations[len(s.Ctx.Observations)-1]
	if field(obs, "tool") != "n4_read_alarms" || field(obs, "rows") != 50 {
		t.Errorf("observation %s", pyjson.Repr(obs))
	}
	out = callOK(t, alarmServer(t, 3), "n4_read_alarms", "limit", pyjson.IntLiteral("1000"))
	if field(out, "row_count") != 3 || field(out, "truncated") != false {
		t.Errorf("3 records: %s", pyjson.Repr(out))
	}
}

func TestReadAlarmsSendsEveryFilter(t *testing.T) {
	s := alarmServer(t, 1)
	callOK(t, s, "n4_read_alarms", "source", "station:|slot:/Drivers/Net", "alarm_class", "defaultAlarmClass",
		"source_state", "offnormal", "ack_state", "unacked", "since", "2026-11-15T00:00:00.000-06:00",
		"until", "2026-11-15T12:00:00Z")
	where := "(source like 'slot:/Drivers/Net' or source like 'slot:/Drivers/Net/*' or " +
		"source like 'local:*slot:/Drivers/Net' or source like 'local:*slot:/Drivers/Net/*') and alarmClass = 'defaultAlarmClass' and " +
		"sourceState = 'offnormal' and ackState = 'unacked' and timestamp.millis >= 1794722400000 and " +
		"timestamp.millis <= 1794744000000"
	if got := sent(s)[0]; got != alarmSelect+" where "+where+alarmOrder {
		t.Errorf("ORD sent %q", got)
	}
	for _, ord := range sent(s) { // the ORD separator inside the bql part cuts the query (HTTP 400)
		if part := strings.TrimSuffix(strings.TrimPrefix(ord, "alarm:|bql:"), "|view:file:ITableToCsv"); strings.Contains(part, "|") {
			t.Errorf("the bql part of %q holds a '|'", ord)
		}
	}
	s = alarmServer(t, 1)
	callOK(t, s, "n4_read_alarms", "ack_state", "acked")
	if got := sent(s)[0]; got != alarmSelect+" where ackState = 'acked'"+alarmOrder {
		t.Errorf("a state filter keeps the open clause: %q", got)
	}
	s = alarmServer(t, 1)
	out := callOK(t, s, "n4_read_alarms", "open_only", false)
	if got := sent(s)[0]; got != alarmSelect+alarmOrder || field(out, "where") != "" {
		t.Errorf("open_only false: %q", got)
	}
}

func TestReadAlarmsRefusesABadFilterBeforeAnyRequest(t *testing.T) {
	for _, c := range []struct {
		kv   []any
		want string
	}{
		{[]any{"source_state", "Offnormal"}, "source_state must be one of normal, offnormal, fault, alert"},
		{[]any{"ack_state", "acked' or '1'='1"}, "ack_state must be one of acked, unacked, ackPending"},
		{[]any{"alarm_class", "x' or alarmClass != '"}, "alarm_class must be an alarm class name"},
		{[]any{"source", "station:|slot:/Drivers/N*"}, "source must be a component ORD"},
		{[]any{"source", "/Drivers/Net"}, "source must be a component ORD"},
		{[]any{"source", "station:|slot:/Drivers/x'y"}, "source must be a component ORD"},
		{[]any{"since", "yesterday"}, "since must be a BAbsTime"},
		{[]any{"since", "2026-11-15T12:00:00Z", "until", "2026-11-15T00:00:00Z"}, "since must not be later than until"},
	} {
		s := alarmServer(t, 1)
		text := callFail(t, s, "n4_read_alarms", c.kv...)
		if len(s.Station.OrdRequests) != 0 || !strings.Contains(text, c.want) {
			t.Errorf("%v: %s (%d requests)", c.kv, text, len(s.Station.OrdRequests))
		}
	}
	args, params, msg := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	args.Set("limit", pyjson.IntLiteral("1001"))
	params.Set("name", "n4_read_alarms")
	params.Set("arguments", args)
	msg.Set("id", 1)
	msg.Set("method", "tools/call")
	msg.Set("params", params)
	if reply := pyjson.Repr(alarmServer(t, 1).Dispatch(msg)); !strings.Contains(reply, "argument limit must be <= 1000") {
		t.Errorf("limit 1001: %s", reply)
	}
}

func TestReadAlarmsSaysWhenTheCountFailedAndPassesAnAuthError(t *testing.T) {
	s := alarmServer(t, 2)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "select count(*)", Code: 400}}, s.Station.OrdRoutes...)
	out := callOK(t, s, "n4_read_alarms")
	note, _ := field(out, "total_note").(string)
	if v, ok := out.Get("total"); !ok || v != nil || !strings.Contains(note, "HTTP 400") || field(out, "row_count") != 2 {
		t.Errorf("reply %s", pyjson.Repr(out))
	}
	s = alarmServer(t, 2)
	s.Station.OrdRoutes = append([]readtoolstest.OrdRoute{{Needle: "alarm:", Code: 403}}, s.Station.OrdRoutes...)
	if text := callFail(t, s, "n4_read_alarms"); !strings.Contains(text, "auth") && !strings.Contains(text, "401") {
		t.Errorf("auth error: %s", text)
	}
}
