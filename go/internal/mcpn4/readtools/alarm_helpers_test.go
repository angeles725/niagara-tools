package readtools_test

// Helpers of the alarm read tests: the station's recorded answers and the calls.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	alarmSelect = "alarm:|bql:select timestamp, uuid, source, sourceState, ackState, alarmClass, priority, alarmData.msgText"
	alarmCount  = "alarm:|bql:select count(*)"
	alarmOrder  = " order by timestamp desc|view:file:ITableToCsv"
	alarmOpen   = "(ackState != 'acked' or sourceState != 'normal')"
)

// alarmRecords is an ITableToCsv answer of n alarm records, newest first, as a 4.15 station
// renders them (display text, localized headers).
func alarmRecords(n int) string {
	var b strings.Builder
	b.WriteString("Timestamp,Uuid,Source,Source State,Ack State,Alarm Class,Priority,Msg Text\n")
	for i := n; i > 0; i-- {
		fmt.Fprintf(&b, "15 nov 2026 18:%02d:58 CST,aa110470-0000-0000-0000-%012d,local:|station:|slot:/Drivers/Net/D%d,"+
			"Offnormal,Unacked,defaultAlarmClass,255,\"Ping Failed, retry\"\n", i, i, i)
	}
	return b.String()
}

// alarmServer is a connected server whose station answers the records and their count.
func alarmServer(t *testing.T, n int) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "select count(*)", Body: fmt.Sprintf("count(toString)\n%d\n", n)},
		{Needle: "order by timestamp desc", Body: alarmRecords(n)},
	}
	return s
}

func callOK(t *testing.T, s *readtoolstest.Server, tool string, kv ...any) *pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, tool, kv...)
	if isErr {
		t.Fatalf("%s: %s", tool, text)
	}
	return out
}

func callFail(t *testing.T, s *readtoolstest.Server, tool string, kv ...any) string {
	t.Helper()
	text, isErr, _ := s.Call(t, tool, kv...)
	if !isErr {
		t.Fatalf("%s succeeded: %s", tool, text)
	}
	return text
}
