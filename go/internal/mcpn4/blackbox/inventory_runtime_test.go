package blackbox

// n4_inventory_diff on the slots a live 4.15 station flags (review of #803 on Prueba_4_15): the
// contracts below are the real ones (organized/, vineflower decompile of 4.15 control-rt and
// history-rt), as Flags.encodeToString letters (r readonly 1, t transient 2, s summary 8, o 256).

import (
	"testing"
)

// liveContracts: BNumericPoint.out 267, BNumericWritable.in1 1, in10 10, fallback 0
// (BNumericPoint.java:31, BNumericWritable.java:138,147,154); BHistoryExt.lastRecord and status 3
// (history-rt BHistoryExt.java:117,125).
func liveContracts() map[string]any {
	return map[string]any{
		"control:NumericWritable": map[string]any{"c": []any{
			map[string]any{"st": "p", "n": "out", "f": "rtso"},
			map[string]any{"st": "p", "n": "in1", "f": "r"},
			map[string]any{"st": "p", "n": "in10", "f": "ts"},
			map[string]any{"st": "p", "n": "fallback"}}},
		"history:NumericIntervalHistoryExt": map[string]any{"c": []any{
			map[string]any{"st": "p", "n": "status", "f": "rt"},
			map[string]any{"st": "p", "n": "lastRecord", "f": "rt"},
			map[string]any{"st": "p", "n": "historyName"}}},
	}
}

func trendRecord(timestamp, value string) map[string]any {
	return map[string]any{"nm": "p", "t": "history:NumericTrendRecord", "s": []any{
		map[string]any{"nm": "p", "n": "timestamp", "t": "baja:AbsTime", "v": timestamp},
		map[string]any{"nm": "p", "n": "value", "t": "baja:Double", "v": value}}}
}

func TestInventoryDiffReportsAStatusConfigSlotAndNotTheRuntimeOnes(t *testing.T) {
	r := inventoryRig(t)
	r.Station.SetRegistry(liveContracts())
	src := r.AddWithOut("Src", 1)
	nw := r.Add("3", "NW", "control:NumericWritable")
	for slot, v := range map[string]float64{"fallback": 21, "out": 21, "in10": 5, "in1": 0} {
		r.Set(nw.Handle, slot, StatusNumeric(v, "0"))
	}
	r.Link(src, "out", nw, "in1") // a linked input is driven by its source: runtime
	hist := r.Add(nw.Handle, "NumericInterval", "history:NumericIntervalHistoryExt")
	r.addProp(hist.Handle, "historyName", "baja:String", "NW", "")
	r.Set(hist.Handle, "status", map[string]any{"nm": "p", "t": "baja:Status", "v": "0"})
	r.Set(hist.Handle, "lastRecord", trendRecord("2026-11-17T04:55:06.000-06:00", "21.0"))
	a := r.snapshotName()
	r.Set(nw.Handle, "fallback", StatusNumeric(33.5, "0"))
	r.Set(nw.Handle, "out", StatusNumeric(33.5, "0"))
	r.Set(nw.Handle, "in10", StatusNumeric(6, "0"))
	r.Set(nw.Handle, "in1", StatusNumeric(7, "0"))
	r.Set(hist.Handle, "status", map[string]any{"nm": "p", "t": "baja:Status", "v": "2"})
	r.Set(hist.Handle, "lastRecord", trendRecord("2026-11-17T05:13:30.000-06:00", "33.5"))
	b := r.snapshotName()
	out := r.Srv.OK("n4_inventory_diff", Args{"a": a, "b": b})
	if got := asJSONText(t, out["changed"]); got != `[{"after":"33.5 {0}","before":"21.0 {0}","path":"slot:/Folder/NW","slot":"fallback"}]` {
		t.Errorf("changed = %s", got)
	}
	if got := asJSONText(t, out["counts"]); got != `{"added":0,"changed":1,"handle_changed":0,"removed":0,"type_changed":0,"unchanged":4}` { // Folder, Src, NW.proxyExt, NumericInterval: no lastRecord
		t.Errorf("counts = %s (added %v removed %v)", got, out["added"], out["removed"])
	}
}
