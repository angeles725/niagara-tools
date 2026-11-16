package blackbox

// n4_bql_query over the alarm: and history: bases through the real server (retro 2026-11-16
// M1): the same /ord/ route and CSV view; the station base answer is unchanged.

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

func TestBqlQueryReadsTheAlarmAndHistoryBases(t *testing.T) {
	r, sv := servletRig(t, nil)
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "alarm:|bql:", Body: []byte("Timestamp,Source State\n16 nov 2026 00:08:56 CST,Offnormal\n")},
		boxstation.OrdRoute{Needle: "history:/" + StationName + "/NW|bql:", Body: []byte("Timestamp,Value\n16 nov 2026 00:00:00 CST,21.5\n")},
		boxstation.OrdRoute{Needle: "driver:Device", Body: kitFixture(t, "bql_devices.csv")})
	alarms := r.Srv.OK("n4_bql_query", Args{"base": "alarm:", "query": "select timestamp, sourceState"})
	history := r.Srv.OK("n4_bql_query", Args{"base": "history:/" + StationName + "/NW", "query": "select timestamp, value"})
	if alarms["row_count"] != float64(1) || history["row_count"] != float64(1) {
		t.Errorf("alarm %v, history %v", alarms, history)
	}
	devices := r.Srv.OK("n4_bql_query", Args{"base": "/Drivers", "query": bqlQuery})
	if devices["row_count"] != float64(5) {
		t.Errorf("station base %v", devices)
	}
	ords := ordRequests(sv)
	if len(ords) != 3 || !strings.HasPrefix(ords[0], "alarm:|bql:") || !strings.HasPrefix(ords[1], "history:/") ||
		!strings.HasPrefix(ords[2], "station:|slot:/Drivers|bql:") {
		t.Errorf("ords %q", ords)
	}
	if text := r.Srv.Err("n4_bql_query", Args{"base": "alarm:", "query": "delete *"}); !strings.Contains(text, "select") {
		t.Errorf("non-select: %q", text)
	}
	if len(ordRequests(sv)) != 3 {
		t.Errorf("a refused query reached the station")
	}
}
