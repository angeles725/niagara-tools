package blackbox

// n4_export_history (station-audits S2): a range of one or more histories written to one private
// CSV file under the state dir, for client reports, from a fake station's answers.

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

// exportSummaryBody is the machine-encoded summary read of a history, newest first.
const (
	exportTempBody = "Encode To String,Encode To String,Status\n" +
		"2026-11-16T02:22:00.000-06:00,21.5,{ok}\n" +
		"2026-11-16T02:21:00.000-06:00,21.25,\"{fault,stale}\"\n" +
		"2026-11-16T02:20:00.000-06:00,21.0,{ok}\n"
	exportDoorBody = "Encode To String,Encode To String,Status\n" +
		"2026-11-16T02:21:00.000-06:00,true,{ok}\n" +
		"2026-11-16T02:20:00.000-06:00,false,{ok}\n"
)

// exportRig is a connected rig with a private state dir whose station answers the summary read
// of /FakeStation/Temp and /FakeStation/Door.
func exportRig(t *testing.T) (*Rig, *boxstation.Servlets, string) {
	t.Helper()
	state := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(state, 0o700); err != nil {
		t.Fatal(err)
	}
	r, sv := servletRig(t, func(o *Options) { o.StateDir = state })
	routeOrds(sv,
		boxstation.OrdRoute{Needle: "history:/FakeStation/Temp", Body: []byte(exportTempBody)},
		boxstation.OrdRoute{Needle: "history:/FakeStation/Door", Body: []byte(exportDoorBody)})
	return r, sv, state
}

func TestExportHistoryWritesOnePrivateCsvUnderTheStateDir(t *testing.T) {
	r, sv, state := exportRig(t)
	out := r.Srv.OK("n4_export_history", Args{"history_ids": []any{"/FakeStation/Temp", "Door"}, "start": "24h"})
	path, _ := out["path"].(string)
	dir := filepath.Join(state, "exports")
	if filepath.Dir(path) != dir || !strings.HasPrefix(filepath.Base(path), StationName+"-") || !strings.HasSuffix(path, ".csv") {
		t.Fatalf("path = %q, want %s/%s-<UTC time>.csv", path, dir, StationName)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v (%v), want 0600", info, err)
	}
	if d, err := os.Stat(dir); err != nil || d.Mode().Perm() != 0o700 {
		t.Errorf("exports dir mode = %v (%v), want 0700", d, err)
	}
	data, _ := os.ReadFile(path)
	want := "history_id,timestamp,value,status\r\n" +
		"/FakeStation/Temp,2026-11-16T02:20:00.000-06:00,21.0,{ok}\r\n" +
		"/FakeStation/Temp,2026-11-16T02:21:00.000-06:00,21.25,\"{fault,stale}\"\r\n" +
		"/FakeStation/Temp,2026-11-16T02:22:00.000-06:00,21.5,{ok}\r\n" +
		"/FakeStation/Door,2026-11-16T02:20:00.000-06:00,false,{ok}\r\n" +
		"/FakeStation/Door,2026-11-16T02:21:00.000-06:00,true,{ok}\r\n"
	if string(data) != want {
		t.Errorf("file =\n%q\nwant\n%q", data, want)
	}
	per := objects(out["per_history"])
	wantPer := []map[string]any{{"history_id": "/FakeStation/Temp", "rows": 3.0, "truncated": false},
		{"history_id": "/FakeStation/Door", "rows": 2.0, "truncated": false}}
	if !reflect.DeepEqual(per, wantPer) {
		t.Errorf("per_history = %v", per)
	}
	if out["rows_total"] != 5.0 || out["bytes"] != float64(len(data)) {
		t.Errorf("rows_total %v bytes %v (file %d)", out["rows_total"], out["bytes"], len(data))
	}
	reqs := ordRequests(sv)
	if len(reqs) != 2 || !strings.Contains(reqs[0], "value.encodeToString") || !strings.Contains(reqs[0], "?start=") {
		t.Errorf("requests = %q", reqs)
	}
}

func TestExportHistoryMarksATruncatedHistory(t *testing.T) {
	r, _, _ := exportRig(t)
	out := r.Srv.OK("n4_export_history", Args{"history_ids": []any{"/FakeStation/Temp"}, "max_rows": 2})
	per := objects(out["per_history"])
	if len(per) != 1 || per[0]["rows"] != 2.0 || per[0]["truncated"] != true || out["rows_total"] != 2.0 {
		t.Errorf("per_history = %v rows_total %v", per, out["rows_total"])
	}
}

func TestExportHistoryRefusesABadIdOrListBeforeAnyRequest(t *testing.T) {
	r, sv, state := exportRig(t)
	for _, ids := range []any{
		[]any{"/FakeStation/Temp", "/FakeStation/../etc"},
		[]any{"/FakeStation/Te\"mp"},
		[]any{"../../etc/passwd"},
		[]any{},
		[]any{3},
		make([]any, 21),
	} {
		res := r.Srv.Call("n4_export_history", Args{"history_ids": ids})
		if !res.IsError {
			t.Errorf("%v: accepted: %v", ids, res.Raw)
		}
	}
	if n := len(ordRequests(sv)); n != 0 {
		t.Errorf("station requests = %d, want 0", n)
	}
	if _, err := os.Stat(filepath.Join(state, "exports")); !os.IsNotExist(err) {
		t.Errorf("exports dir exists after refusals (%v)", err)
	}
}
