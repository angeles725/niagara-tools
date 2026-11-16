package readtools_test

// Helpers of the n4_event_report tests: two alarm events of one source (the newer still active,
// the older back to normal and acknowledged), the history extension beside the alarm extension
// and the history records of each event window.

import (
	"testing"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const (
	eventSelect = "alarm:|bql:select timestamp.encodeToString, uuid, source, sourceState.encodeToString, " +
		"ackState.encodeToString, alarmClass, alarmTransition.encodeToString, normalTime.encodeToString, " +
		"ackTime.encodeToString, user"
	eventSource = "(source like 'slot:/Plant/Comp1/AmpsAlarm' or source like 'slot:/Plant/Comp1/AmpsAlarm/*' or " +
		"source like 'local:*slot:/Plant/Comp1/AmpsAlarm' or source like 'local:*slot:/Plant/Comp1/AmpsAlarm/*')"
	eventExtOrd = "station:|slot:/Plant/Comp1|bql:select slotPath, historyConfig.id from history:HistoryExt|view:file:ITableToCsv"
	// The windows of the two events (margin 15 minutes), sent as UTC.
	windowActive = "history:/FakeStation/Comp1_Amps?start=2026-11-16T15:45:00.000Z;end=2026-11-16T17:00:00.000Z" + summaryQuery
	windowClosed = "history:/FakeStation/Comp1_Amps?start=2026-11-16T13:45:00.000Z;end=2026-11-16T14:45:00.000Z" + summaryQuery
)

// eventRecords is the events read: newest first, the active one (no normal time) then the one
// back to normal and acknowledged by admin.
const eventRecords = "Encode To String,Uuid,Source,Encode To String,Encode To String,Alarm Class,Encode To String," +
	"Encode To String,Encode To String,User\n" +
	"2026-11-16T10:00:00.000-06:00,aa000000-0000-0000-0000-000000000002,local:|station:|slot:/Plant/Comp1/AmpsAlarm," +
	"offnormal,unacked,defaultAlarmClass,offnormal,1969-12-31T18:00:00.000-06:00,1969-12-31T18:00:00.000-06:00,Unknown User\n" +
	"2026-11-16T08:00:00.000-06:00,aa000000-0000-0000-0000-000000000001,local:|station:|slot:/Plant/Comp1/AmpsAlarm," +
	"normal,acked,defaultAlarmClass,offnormal,2026-11-16T08:30:00.000-06:00,2026-11-16T08:40:00.000-06:00,admin\n"

// eventExts is the extension lookup at the point: the sibling history extension, and one deeper
// that belongs to another component.
const eventExts = "Slot Path,History Config Id\n" +
	"slot:/Plant/Comp1/Sub/NumericCov,/FakeStation/Deep\nslot:/Plant/Comp1/NumericCov,/FakeStation/Comp1_Amps\n"

// windowRecords are the history records of a window, newest first.
const (
	activeRecords = "Encode To String,Encode To String,Status\n" +
		"2026-11-16T10:50:00.000-06:00,30.5,{ok}\n2026-11-16T10:20:00.000-06:00,41.2,{ok}\n2026-11-16T09:50:00.000-06:00,12.0,{ok}\n"
	closedRecords = "Encode To String,Encode To String,Status\n" +
		"2026-11-16T08:40:00.000-06:00,11.0,{ok}\n2026-11-16T08:10:00.000-06:00,38.75,{ok}\n2026-11-16T07:50:00.000-06:00,10.0,{ok}\n"
)

// eventServer is a connected server whose station answers the events, their count, the
// extension lookup (exts) and both windows, at 2026-11-16T11:00-06:00.
func eventServer(t *testing.T, exts string) *readtoolstest.Server {
	t.Helper()
	now := time.Date(2026, 11, 16, 17, 0, 0, 0, time.UTC)
	saved := bql.Now
	bql.Now = func() time.Time { return now }
	t.Cleanup(func() { bql.Now = saved })
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{})
	s.Connect(t)
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{
		{Needle: "alarmTransition.encodeToString", Body: eventRecords},
		{Needle: "alarm:|bql:select count(*)", Body: "count(toString)\n2\n"},
		{Needle: "history:HistoryExt", Body: exts},
		{Needle: "start=2026-11-16T15:45", Body: activeRecords},
		{Needle: "start=2026-11-16T13:45", Body: closedRecords},
	}
	return s
}

// events is the report's events.
func events(t *testing.T, out *pyjson.Object) []*pyjson.Object {
	t.Helper()
	raw, _ := field(out, "events").([]any)
	evs := make([]*pyjson.Object, len(raw))
	for i, e := range raw {
		evs[i] = e.(*pyjson.Object)
	}
	return evs
}
