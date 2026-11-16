package readtools_test

// n4_list_alarm_classes through the in-memory station: the classes decoded, and the recipients
// found from the links into their routeAlarm action.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	classesOrd    = "station:|slot:/|bql:select slotPath, name, priority, ackRequired, totalAlarmCount, openAlarmCount, inAlarmCount, unackedAlarmCount from alarm:AlarmClass|view:file:ITableToCsv"
	recipientsOrd = "station:|slot:/|bql:select slotPath, type from alarm:AlarmRecipient|view:file:ITableToCsv"
	servicePath   = "/Services/AlarmService"
)

// nodes is a LoadTree answer: path, then key/value pairs of its node, per entry.
func nodes(entries ...[]any) *pyjson.Object {
	out := pyjson.NewObject()
	for _, e := range entries {
		n := pyjson.NewObject()
		for i := 1; i < len(e); i += 2 {
			n.Set(e[i].(string), e[i+1])
		}
		out.Set(e[0].(string), n)
	}
	return out
}

// routeLink is the nodes of one link named name from sourceOrd's sourceSlot to targetSlot.
func routeLink(name, sourceOrd, sourceSlot, targetSlot string) [][]any {
	return [][]any{{name, "t", "baja:Link"}, {name + "/sourceOrd", "v", sourceOrd},
		{name + "/sourceSlotName", "v", sourceSlot}, {name + "/targetSlotName", "v", targetSlot}}
}

func classesServer(t *testing.T) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "from alarm:AlarmClass", Body: "Slot Path,Name,Priority,Ack Required,Total Alarm Count,Open Alarm Count," +
			"In Alarm Count,Unacked Alarm Count\n" +
			"slot:/Services/AlarmService/defaultAlarmClass,defaultAlarmClass,toOffnormal=255;toFault=255;toNormal=255;toAlert=255;," +
			"{toOffnormal toFault toNormal},25,22,3,20\n" +
			"slot:/Services/AlarmService/Critical,Critical,weird,{alert toOffnormal},1,1,1,x\n"},
		{Needle: "from alarm:AlarmRecipient", Body: "Slot Path,Type\n" +
			"slot:/Services/AlarmService/Console,alarm:ConsoleRecipient\nslot:/Services/AlarmService/Mail,email:EmailRecipient\n"},
	}
	console := append([][]any{{"", "h", "r1"}}, routeLink("Link", "h:c1", "alarm", "routeAlarm")...)
	console = append(console, routeLink("Link1", "station:|h:c2", "alarm", "routeAlarm")...)
	console = append(console, routeLink("Link2", "h:zz", "alarm", "routeAlarm")...)
	console = append(console, routeLink("Ack", "h:c1", "alarm", "routeAlarmAck")...)
	mail := append([][]any{{"", "h", "r2"}}, routeLink("Link", "station:|slot:/Services/AlarmService/Critical", "alarm", "routeAlarm")...)
	s.Station.Trees = map[string]*pyjson.Object{
		"station:|slot:" + servicePath: nodes([]any{"", "h", "s1"}, []any{"defaultAlarmClass", "h", "c1", "t", "alarm:AlarmClass"},
			[]any{"Critical", "h", "c2", "t", "alarm:AlarmClass"}, []any{"Console", "h", "r1"}),
		"station:|slot:" + servicePath + "/Console": nodes(console...),
		"station:|slot:" + servicePath + "/Mail":    nodes(mail...),
	}
	return s
}

func TestListAlarmClassesDecodesEachClassAndItsRecipients(t *testing.T) {
	s := classesServer(t)
	before := s.Station.Requests
	out := callOK(t, s, "n4_list_alarm_classes")
	if got := sent(s); strings.Join(got, "\n") != classesOrd+"\n"+recipientsOrd {
		t.Errorf("ORDs sent %q", got)
	}
	if loads := s.Station.Requests - before; loads != 3 { // the class folder and two recipients
		t.Errorf("%d tree loads", loads)
	}
	classes := field(out, "classes").([]any)
	want := []string{
		"{'path': '/Services/AlarmService/defaultAlarmClass', 'name': 'defaultAlarmClass', " +
			"'priority': {'toOffnormal': 255, 'toFault': 255, 'toNormal': 255, 'toAlert': 255}, " +
			"'ack_required': {'toOffnormal': True, 'toFault': True, 'toNormal': True, 'toAlert': False}, " +
			"'total_alarm_count': 25, 'open_alarm_count': 22, 'in_alarm_count': 3, 'unacked_alarm_count': 20, " +
			"'recipients': [{'path': '/Services/AlarmService/Console', 'type': 'alarm:ConsoleRecipient', 'link_name': 'Link'}]}",
		"{'path': '/Services/AlarmService/Critical', 'name': 'Critical', 'priority': 'weird', " +
			"'ack_required': {'toOffnormal': True, 'toFault': False, 'toNormal': False, 'toAlert': True}, " +
			"'total_alarm_count': 1, 'open_alarm_count': 1, 'in_alarm_count': 1, 'unacked_alarm_count': 'x', " +
			"'recipients': [{'path': '/Services/AlarmService/Console', 'type': 'alarm:ConsoleRecipient', 'link_name': 'Link1'}, " +
			"{'path': '/Services/AlarmService/Mail', 'type': 'email:EmailRecipient', 'link_name': 'Link'}]}",
	}
	if len(classes) != 2 {
		t.Fatalf("classes %s", pyjson.Repr(out))
	}
	for i, w := range want {
		if got := pyjson.Repr(classes[i]); got != w {
			t.Errorf("class %d\n got %s\nwant %s", i, got, w)
		}
	}
	if got := pyjson.Repr(field(out, "recipients")); got != "[{'path': '/Services/AlarmService/Console', "+
		"'type': 'alarm:ConsoleRecipient', 'classes': ['defaultAlarmClass', 'Critical']}, "+
		"{'path': '/Services/AlarmService/Mail', 'type': 'email:EmailRecipient', 'classes': ['Critical']}]" {
		t.Errorf("recipients %s", got)
	}
	if got := pyjson.Repr(field(out, "unresolved_links")); got != "[{'recipient_path': '/Services/AlarmService/Console', "+
		"'link_name': 'Link2', 'source_ord': 'h:zz'}]" || field(out, "recipients_truncated") != false {
		t.Errorf("unresolved %s, truncated %v", got, field(out, "recipients_truncated"))
	}
}

func TestListAlarmClassesReadsAtMostFiftyRecipients(t *testing.T) {
	s := classesServer(t)
	body := "Slot Path,Type\n"
	for i := 0; i < 51; i++ {
		body += fmt.Sprintf("slot:/Services/AlarmService/R%d,alarm:ConsoleRecipient\n", i)
	}
	s.Station.OrdRoutes[1].Body = body
	before := s.Station.Requests
	out := callOK(t, s, "n4_list_alarm_classes")
	if loads := s.Station.Requests - before; loads != 51 || field(out, "recipients_truncated") != true ||
		len(field(out, "recipients").([]any)) != 50 {
		t.Errorf("%d loads, reply %v", loads, field(out, "recipients_truncated"))
	}
}
