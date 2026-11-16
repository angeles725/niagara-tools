package writeops_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/writeops"
)

const (
	ptBool = "station:|slot:/Drivers/BacnetNetwork/$32004/points/Fan_S"
	ptEnum = "station:|slot:/Drivers/BacnetNetwork/$32004/points/Mode"
)

func TestAddHistoryPicksTheExtensionTypeAndNameFromThePointType(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptBool, typedPoint("control:BooleanWritable", "12"),
		ptEnum, typedPoint("control:EnumWritable", "13"))
	p, err := planHistory(c, histArgs("points", []any{ptA, ptBool, ptEnum}))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range [][2]string{{"NumericInterval", "history:NumericIntervalHistoryExt"},
		{"BooleanInterval", "history:BooleanIntervalHistoryExt"}, {"EnumInterval", "history:EnumIntervalHistoryExt"}} {
		op := js(t, p.Ops[i])
		if !strings.Contains(op, `"n": "`+want[0]+`", "b": {"nm": "p", "t": "`+want[1]+`"`) {
			t.Errorf("op %d = %s, want %v", i, op, want)
		}
		if inv := js(t, p.Inverse[i]); !strings.Contains(inv, `"n": "`+want[0]+`"`) {
			t.Errorf("inverse %d = %s", i, inv)
		}
	}
	named, err := planHistory(c, histArgs("points", []any{ptBool}, "ext_name", "Trend"))
	if err != nil || !strings.Contains(js(t, named.Ops[0]), `"n": "Trend", "b": {"nm": "p", "t": "history:BooleanIntervalHistoryExt"`) {
		t.Errorf("ext_name on a boolean point: err = %v, ops %v", err, named)
	}
}

func TestAddHistoryTypeRefusals(t *testing.T) {
	folder := "station:|slot:/Drivers/BacnetNetwork/$32004/points"
	c := histPoints(t, ptA, point("10"), ptBool, typedPoint("control:BooleanWritable", "12"),
		folder, typedPoint("baja:Folder", "14"))
	_, err := planHistory(c, histArgs("points", []any{ptBool}, "ext_type", "history:NumericIntervalHistoryExt"))
	wantRefusal(t, "mismatched ext_type", err, "ext_type history:NumericIntervalHistoryExt does not fit "+ptBool+
		", a boolean point (control:BooleanWritable): use history:BooleanIntervalHistoryExt or leave ext_type out")
	_, err = planHistory(c, histArgs("points", []any{folder}))
	wantRefusal(t, "not a point", err, folder+" is a baja:Folder, not a numeric, boolean, enum or string control point")
	if _, err = planHistory(c, histArgs("points", []any{ptA}, "ext_type", "history:NumericIntervalHistoryExt")); err != nil {
		t.Errorf("a matching ext_type: %v", err)
	}
}

func TestAddHistoryRefusesAHistoryIdTheStationAlreadyHas(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	c.ids = "Slot Path,Id\nslot:/Folder/Old/NumericInterval,/FakeStation/S2005_RA_T\n"
	_, err := planHistory(c, histArgs())
	wantRefusal(t, "station id", err, "history id '/FakeStation/S2005_RA_T' of "+ptB+
		" is already used by the extension at slot:/Folder/Old/NumericInterval")
	if len(c.ords) != 1 || !strings.HasPrefix(c.ords[0], "station:|slot:/|bql:select slotPath, historyConfig.id from history:HistoryExt") {
		t.Errorf("ords read = %v, want one station-wide history id query", c.ords)
	}
	c.ids = "Slot Path,Id\nslot:/Folder/Old/NumericInterval,/OtherStation/S2005_RA_T\n"
	if _, err = planHistory(c, histArgs()); err != nil {
		t.Errorf("an id of another station: %v", err)
	}
	c.ids, c.idsErr = "", errNoServlet
	_, err = planHistory(c, histArgs())
	wantRefusal(t, "unreadable ids", err, "could not read the station's history ids to check for a repeat")
	c.ords = nil
	if _, err = planHistory(c, histArgs("history_name", "%parent.displayName%")); err != nil || len(c.ords) != 0 {
		t.Errorf("no id to check: err = %v, ords %v", err, c.ords)
	}
}

func TestAddHistoryCapacityNoteFollowsTheInterval(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	for ms, want := range map[string]string{"100": "about 50 s at 100 ms", "300000": "about 41.7 h at 5 min",
		"900000": "about 5.2 d at 15 min"} {
		p, err := planHistory(c, histArgs("interval_ms", pyjson.IntLiteral(ms)))
		if err != nil {
			t.Fatal(err)
		}
		note := js(t, p.Notes)
		if !strings.Contains(note, "Capacity stays at the extension default (500 records, Roll): "+want+".") {
			t.Errorf("interval %s ms: notes %s, want %q", ms, note, want)
		}
	}
}

func TestAddHistoryReadbackChecksTheAssignedHistoryIdAndFault(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	args := histArgs()
	p, err := planHistory(c, args)
	if err != nil {
		t.Fatal(err)
	}
	replies := []any{[]any{obj("nn", "NumericInterval")}, []any{obj("nn", "NumericInterval")}}
	inverse, _ := writeops.AddHistory.Inverse(p, replies)
	ok := doneExt("NumericInterval", "true", "S2004_RA_T")
	for name, b := range map[string]*pyjson.Object{
		"duplicate fault": doneExt("NumericInterval", "true", "S2005_RA_T", "faultCause",
			obj("t", "baja:String", "v", "Duplicate history id: /FakeStation/S2005_RA_T")),
		"other id": doneExt("NumericInterval", "true", "S2005_RA_T_1"),
	} {
		c.nodes = obj(ptA, ok, ptB, b)
		requested, _, observed, verdict, err := writeops.AddHistory.Readback(c, args, p, replies, inverse)
		if err != nil || verdict != "mismatch" {
			t.Errorf("%s: verdict = %s, err = %v, observed %s", name, verdict, err, js(t, observed))
		}
		if want := `"historyId": "/FakeStation/S2005_RA_T", "faultCause": ""`; !strings.Contains(js(t, requested), want) {
			t.Errorf("%s: requested %s, want %s", name, js(t, requested), want)
		}
	}
	c.nodes = obj(ptA, ok, ptB, doneExt("NumericInterval", "true", "S2005_RA_T"))
	if _, _, observed, verdict, _ := writeops.AddHistory.Readback(c, args, p, replies, inverse); verdict != "verified" {
		t.Errorf("the predicted id and no fault: verdict = %s, observed %s", verdict, js(t, observed))
	}
}
