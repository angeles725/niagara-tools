package blackbox

// n4_read_history window, status filter and history type (mcp-n4 delta M2): from/to bound the
// range the station answers, status keeps only the matching records on the station side, and
// history_type=true states whether the history records on change of value or every interval.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

const (
	hwRecords = "Timestamp,Value,Status\n16 nov 2026 02:25:05 CST,\"20,3\",{ok}\n16 nov 2026 02:25:00 CST,\"20,3\",{fault}\n"
	hwExts    = "Slot Path,History Config Id,Type,Encode To String\n" +
		"slot:/Unit/Setpoint/hist,/FakeStation/Setpoint,history:NumericCovHistoryExt,true:0\n" +
		"slot:/Unit/RoomTemp/hist,/FakeStation/RoomTemp,history:NumericIntervalHistoryExt,false:300000\n"
)

// hwRig is a connected rig whose station answers the records, the count and the history
// extensions with their collection interval.
func hwRig(t *testing.T) (*Rig, *boxstation.Servlets) {
	t.Helper()
	r, sv := servletRig(t, nil)
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "from history:HistoryExt", Body: []byte(hwExts)},
		boxstation.OrdRoute{Needle: "select count(*)", Body: []byte("count(toString)\n2\n")},
		boxstation.OrdRoute{Needle: "order by timestamp desc", Body: []byte(hwRecords)})
	return r, sv
}

func TestReadHistoryFromToBoundsTheRangeTheStationAnswers(t *testing.T) {
	r, sv := hwRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Setpoint",
		"from": "2026-11-16T08:00:00Z", "to": "2026-11-16T09:00:00Z"})
	window := "history:/FakeStation/Setpoint?start=2026-11-16T08:00:00Z;end=2026-11-16T09:00:00Z|bql:"
	reqs := ordRequests(sv)
	if len(reqs) != 2 || !strings.HasPrefix(reqs[0], window+"select timestamp, value, status order by timestamp desc|") ||
		!strings.HasPrefix(reqs[1], window+"select count(*)|") {
		t.Fatalf("station requests = %q, want the records and the count both bounded by %s", reqs, window)
	}
	if out["from"] != "2026-11-16T08:00:00Z" || out["to"] != "2026-11-16T09:00:00Z" || out["order"] != "newest_first" {
		t.Errorf("from %v to %v order %v", out["from"], out["to"], out["order"])
	}
	if out["range"] != "start=2026-11-16T08:00:00Z;end=2026-11-16T09:00:00Z" {
		t.Errorf("range = %v", out["range"])
	}
}

func TestReadHistoryRefusesFromWithStartOrALateFrom(t *testing.T) {
	r, sv := hwRig(t)
	for _, bad := range []Args{
		{"history_id": "/FakeStation/Setpoint", "from": "1h", "start": "2h"},
		{"history_id": "/FakeStation/Setpoint", "from": "2026-11-16T09:00:00Z", "to": "2026-11-16T08:00:00Z"},
		{"history_id": "/FakeStation/Setpoint", "from": "yesterday"},
	} {
		res := r.Srv.Call("n4_read_history", bad)
		if !res.IsError || len(ordRequests(sv)) != 0 {
			t.Errorf("%v: %v, requests %v", bad, res.Raw, ordRequests(sv))
		}
	}
	res := r.Srv.Call("n4_read_history", Args{"history_id": "/FakeStation/Setpoint", "from": "yesterday"})
	if !strings.Contains(res.Text, "from must be") {
		t.Errorf("bad from names start, not from: %s", res.Text)
	}
}

func TestReadHistoryStatusNotOkKeepsOnlyTheNonOkRecords(t *testing.T) {
	r, sv := hwRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Setpoint", "status": "not-ok"})
	reqs := ordRequests(sv)
	if len(reqs) != 2 ||
		!strings.Contains(reqs[0], "|bql:select timestamp, value, status where status.ok = false order by timestamp desc|") ||
		!strings.Contains(reqs[1], "|bql:select count(*) where status.ok = false|") {
		t.Fatalf("station requests = %q, want the records and the count filtered by status.ok = false", reqs)
	}
	if out["status"] != "not-ok" || out["total_records"] != 2.0 {
		t.Errorf("status %v total_records %v", out["status"], out["total_records"])
	}
	if code := r.Srv.CallError("n4_read_history", Args{"history_id": "/FakeStation/Setpoint", "status": "broken"}); code != -32602 ||
		len(ordRequests(sv)) != 2 {
		t.Errorf("unknown status: code %d, requests %d", code, len(ordRequests(sv)))
	}
}

func TestReadHistoryTypeOfACovAndOfAnIntervalHistory(t *testing.T) {
	r, sv := hwRig(t)
	cov := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Setpoint", "history_type": true})
	want := map[string]any{"kind": "cov", "interval": nil, "interval_ms": nil,
		"ext": "slot:/Unit/Setpoint/hist", "ext_type": "history:NumericCovHistoryExt"}
	if !reflect.DeepEqual(cov["history_type"], want) {
		t.Errorf("COV history_type = %v, want %v", cov["history_type"], want)
	}
	interval := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/RoomTemp", "history_type": true})
	want = map[string]any{"kind": "interval", "interval": "5m", "interval_ms": 300000.0,
		"ext": "slot:/Unit/RoomTemp/hist", "ext_type": "history:NumericIntervalHistoryExt"}
	if !reflect.DeepEqual(interval["history_type"], want) {
		t.Errorf("interval history_type = %v, want %v", interval["history_type"], want)
	}
	if n := len(ordRequests(sv)); n != 6 {
		t.Errorf("station requests = %d, want 6 (records, count, one extension read per call)", n)
	}
	gone := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Imported", "history_type": true})
	note, _ := gone["history_type_note"].(string)
	if v, has := gone["history_type"]; !has || v != nil || !strings.Contains(note, "no history extension") {
		t.Errorf("history with no extension: history_type %v note %q", v, note)
	}
}

func TestReadHistoryWithoutTheNewArgumentsAnswersAsBefore(t *testing.T) {
	r, sv := hwRig(t)
	out := r.Srv.OK("n4_read_history", Args{"history_id": "/FakeStation/Setpoint", "limit": 5})
	reqs := ordRequests(sv)
	want := []string{"history:/FakeStation/Setpoint|bql:select timestamp, value, status order by timestamp desc|",
		"history:/FakeStation/Setpoint|bql:select count(*)|"}
	if len(reqs) != 2 || !strings.HasPrefix(reqs[0], want[0]) || !strings.HasPrefix(reqs[1], want[1]) {
		t.Fatalf("station requests = %q, want %q", reqs, want)
	}
	keys := make([]string, 0, len(out))
	for k := range out {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	wantKeys := []string{"columns", "elapsed_ms", "history_id", "order", "row_count", "rows", "total_records", "truncated"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Errorf("reply keys = %q, want %q", keys, wantKeys)
	}
	if out["row_count"] != 2.0 || out["order"] != "newest_first" || out["truncated"] != false || len(out["rows"].([]any)) != 2 {
		t.Errorf("reply = %v", out)
	}
}
