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
