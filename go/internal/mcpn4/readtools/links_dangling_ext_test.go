package readtools_test

import (
	"testing"
)

// extTree: P1 has only an alarm extension, P2 only a history extension, P3 nothing.
const extTree = `{"": {"t": "baja:Folder", "h": "3"},
	"P1": {"t": "control:NumericPoint", "h": "4"}, "P1/out": {"t": "baja:StatusNumeric"},
	"P1/Alarm": {"t": "alarm:OutOfRangeAlarmExt", "h": "5"},
	"P2": {"t": "control:NumericPoint", "h": "6"}, "P2/out": {"t": "baja:StatusNumeric"},
	"P2/Hist": {"t": "history:NumericIntervalHistoryExt", "h": "7"},
	"P3": {"t": "control:NumericPoint", "h": "8"}, "P3/out": {"t": "baja:StatusNumeric"}}`

// An alarm or history extension reads its point's out without a link (OD007, hist-alarms M8).
func TestDanglingOutputsCountAnExtensionAsAConsumer(t *testing.T) {
	s := serve(t, "station:|slot:/Folder", extTree)
	text, isErr, out := s.Call(t, "n4_find_dangling_outputs", "ord", "station:|slot:/Folder")
	if isErr {
		t.Fatal(text)
	}
	if got := marshal(t, lookupOut(out, "dangling")); got != `[{"path": "/Folder/P3", "type": "control:NumericPoint"}]` {
		t.Errorf("dangling %s", got)
	}
	want := `[{"path": "/Folder/P1", "type": "control:NumericPoint", "extension": "alarm:OutOfRangeAlarmExt"}, ` +
		`{"path": "/Folder/P2", "type": "control:NumericPoint", "extension": "history:NumericIntervalHistoryExt"}]`
	if got := marshal(t, lookupOut(out, "consumed_by_extension")); got != want {
		t.Errorf("consumed_by_extension %s", got)
	}
}

func lookupOut(o interface{ Get(string) (any, bool) }, k string) any { v, _ := o.Get(k); return v }

// termTree: W1 is a writable point with a proxy extension (its out drives a device), W2 a writable
// whose frozen proxyExt is the default control:NullProxyExt (it drives nothing), C1 a calendar schedule, S1 a weekly schedule, P3 a plain point; none has a link.
const termTree = `{"": {"t": "baja:Folder", "h": "3"},
	"W1": {"t": "control:NumericWritable", "h": "4"}, "W1/out": {"t": "baja:StatusNumeric"},
	"W1/proxyExt": {"t": "modbusAsync:ModbusAsyncProxyExt", "h": "5"},
	"W2": {"t": "control:NumericWritable", "h": "6"}, "W2/out": {"t": "baja:StatusNumeric"},
	"W2/proxyExt": {"t": "control:NullProxyExt", "h": "10"},
	"C1": {"t": "schedule:CalendarSchedule", "h": "7"}, "C1/out": {"t": "baja:StatusBoolean"},
	"S1": {"t": "schedule:BooleanSchedule", "h": "8"}, "S1/out": {"t": "baja:StatusBoolean"},
	"P3": {"t": "control:NumericPoint", "h": "9"}, "P3/out": {"t": "baja:StatusNumeric"}}`

// Retro 2026-11-15 row 14: a writable that drives a device through its proxy extension, and a
// calendar schedule that schedules read by ORD, are not dangling; they are listed apart with why.
func TestDanglingOutputsListTerminalPointsAndCalendarsApart(t *testing.T) {
	s := serve(t, "station:|slot:/Folder", termTree)
	text, isErr, out := s.Call(t, "n4_find_dangling_outputs", "ord", "station:|slot:/Folder")
	if isErr {
		t.Fatal(text)
	}
	want := `[{"path": "/Folder/W2", "type": "control:NumericWritable"}, ` +
		`{"path": "/Folder/S1", "type": "schedule:BooleanSchedule"}, {"path": "/Folder/P3", "type": "control:NumericPoint"}]`
	if got := marshal(t, lookupOut(out, "dangling")); got != want {
		t.Errorf("dangling %s", got)
	}
	terminal := `[{"path": "/Folder/W1", "type": "control:NumericWritable", "reason": "its out drives a device through its proxy extension"}, ` +
		`{"path": "/Folder/C1", "type": "schedule:CalendarSchedule", "reason": "a calendar schedule is read by other schedules by ORD, never by a link"}]`
	if got := marshal(t, lookupOut(out, "terminal_by_design")); got != terminal {
		t.Errorf("terminal_by_design %s", got)
	}
}

func TestDanglingOutputsOfASubtreeWithNoTerminalPointKeepTheirReply(t *testing.T) {
	s := serve(t, "station:|slot:/Folder", extTree)
	_, _, out := s.Call(t, "n4_find_dangling_outputs", "ord", "station:|slot:/Folder")
	if _, ok := out.Get("terminal_by_design"); ok {
		t.Error("terminal_by_design present with no terminal point")
	}
}
