package bql_test

// Go twins of the test_bql tool tests that replace the HTTP opener or read srv.ctx
// (go_server.go_twin("bql", "injection")): the in-memory station answers GET /ord/ from routes
// like test_bql.FakeOpener.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/registry"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

const devicesQuery = "select slotPath, name, type from driver:Device"

// routes are the recorded answers of the three inventory queries.
func routes(t *testing.T, edit func(string) string) []readtoolstest.OrdRoute {
	if edit == nil {
		edit = func(s string) string { return s }
	}
	return []readtoolstest.OrdRoute{{Needle: "driver:DeviceNetwork", Body: edit(fixture(t, "bql_networks.csv"))},
		{Needle: "driver:Device", Body: edit(fixture(t, "bql_devices.csv"))},
		{Needle: "control:ControlPoint", Body: edit(fixture(t, "bql_points.csv"))}}
}

// bqlServer is a connected server whose station answers the recorded BQL fixtures; stateDir
// is the context's --state-dir ("" leaves it unset).
func bqlServer(t *testing.T, stateDir string) *readtoolstest.Server {
	t.Helper()
	s := readtoolstest.NewServer(readtoolstest.NewStation(), session.Config{StateDir: stateDir})
	s.Connect(t)
	s.Station.OrdRoutes = routes(t, nil)
	return s
}

func ok(t *testing.T, s *readtoolstest.Server, tool string, kv ...any) *pyjson.Object {
	t.Helper()
	text, isErr, out := s.Call(t, tool, kv...)
	if isErr {
		t.Fatalf("%s: %s", tool, text)
	}
	return out
}

func fail(t *testing.T, s *readtoolstest.Server, tool string, kv ...any) string {
	t.Helper()
	text, isErr, _ := s.Call(t, tool, kv...)
	if !isErr {
		t.Fatalf("%s succeeded: %s", tool, text)
	}
	return text
}

func has(t *testing.T, text string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(text, p) {
			t.Errorf("%q lacks %q", text, p)
		}
	}
}

func repr(v any) string { return pyjson.Repr(v) }

func TestQueryReturnsRowsAndColumnsFromThePathFormOrd(t *testing.T) {
	s := bqlServer(t, "")
	out := ok(t, s, "n4_bql_query", "base", "station:|slot:/Drivers", "query", devicesQuery)
	if repr(get(out, "columns")) != "['Slot Path', 'Name', 'Type']" || get(out, "row_count") != 5 || get(out, "truncated") != false {
		t.Errorf("reply %s", repr(out))
	}
	if _, ok := out.Get("elapsed_ms"); !ok {
		t.Error("no elapsed_ms")
	}
	req := s.Station.OrdRequests
	if len(req) != 1 || req[0].Ord != "station:|slot:/Drivers|bql:"+devicesQuery+"|view:file:ITableToCsv" || req[0].Timeout != 60 {
		t.Errorf("requests %+v", req)
	}
}

func TestRowCapAndTimeoutArePassedThrough(t *testing.T) {
	s := bqlServer(t, "")
	out := ok(t, s, "n4_bql_query", "base", "/Drivers", "max_rows", pyjson.IntLiteral("2"), "timeout_s", pyjson.IntLiteral("5"), "query", "select slotPath from control:ControlPoint")
	if get(out, "row_count") != 2 || get(out, "truncated") != true || s.Station.OrdRequests[0].Timeout != 5 {
		t.Errorf("reply %s, requests %+v", repr(out), s.Station.OrdRequests)
	}
}

func TestAByteCapInsideAQuotedCellDropsTheTornRecord(t *testing.T) {
	s := bqlServer(t, "")
	body := "Slot Path,Name\nslot:/Z,\"ok\"\nslot:/A,\"line one\nline two\"\nslot:/B,\"b\"\n"
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "driver:Device", Body: body}}
	s.Station.OrdCap = strings.Index(body, "line two") + 2 // the cap falls inside A's quoted cell
	out := ok(t, s, "n4_bql_query", "base", "/Drivers", "query", "select slotPath, name from driver:Device")
	if repr(get(out, "rows")) != "[{'Slot Path': 'slot:/Z', 'Name': 'ok'}]" || get(out, "truncated") != true {
		t.Errorf("reply %s", repr(out))
	}
}

func TestNonSelectAndPipeInjectionSendNothing(t *testing.T) {
	s := bqlServer(t, "")
	for _, q := range []string{"delete from driver:Device", "select name from driver:Device|view:web:PropertySheet"} {
		has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "query", q), "select")
	}
	fail(t, s, "n4_bql_query", "base", "/Drivers|bql:select x from y", "query", "select a from b")
	if len(s.Station.OrdRequests) != 0 {
		t.Errorf("requests sent: %+v", s.Station.OrdRequests)
	}
}

func TestA401OnTheBqlPathIsActionableAndNotRetried(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "driver:Device", Code: 401}}
	text := fail(t, s, "n4_bql_query", "base", "/Drivers", "query", "select name from driver:Device")
	has(t, text, "HTTPBasicScheme", "do not retry")
	if strings.Contains(text, readtoolstest.Password) || len(s.Station.OrdRequests) != 1 {
		t.Errorf("%q after %d requests", text, len(s.Station.OrdRequests))
	}
}

func TestA400NamesTheQueryAsTheLikelyCause(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "driver:Device", Code: 400}}
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "query", "select name from driver:Device"), "rejected")
}

func TestTheReadIsRecordedAsASessionObservation(t *testing.T) {
	s := bqlServer(t, "")
	ok(t, s, "n4_bql_query", "base", "/Drivers", "query", "select name from driver:Device")
	obs := s.Ctx.Observations[len(s.Ctx.Observations)-1]
	if get(obs, "tool") != "n4_bql_query" || get(obs, "rows") != 5 || get(obs, "ord") != "/Drivers" {
		t.Errorf("observation %s", repr(obs))
	}
	if _, ok := obs.Get("count"); ok { // count means dangling outputs to the retro
		t.Error("count in a read observation")
	}
}

func TestItIsListedAsAReadOnlyTool(t *testing.T) {
	reg, read := registry.Default(), map[string]bool{}
	for _, tool := range reg.Read {
		read[tool.Name] = true
	}
	for _, tool := range reg.Write {
		if tool.Name == "n4_bql_query" {
			t.Error("n4_bql_query is a write tool")
		}
	}
	if !read["n4_bql_query"] || !read["n4_inventory"] {
		t.Errorf("read tools %v", read)
	}
}

func TestTheReadIsAuditedWithOutcomeRead(t *testing.T) {
	dir := t.TempDir()
	s := bqlServer(t, dir) // the write machinery's audit log, as --allow-writes gives it
	readtools.Register(s.Server, &readtools.Tools{Ctx: s.Ctx, NewClient: s.Station.Factory, ServerVersion: "test",
		Audit: journal.NewAuditLog(dir)})
	ok(t, s, "n4_bql_query", "base", "/Drivers", "query", "select name from driver:Device")
	data, err := os.ReadFile(filepath.Join(dir, "audit.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		var e map[string]any
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, e)
	}
	if len(lines) != 1 || lines[0]["tool"] != "n4_bql_query" || lines[0]["outcome"] != "read" ||
		strings.Contains(string(data), readtoolstest.Password) {
		t.Errorf("audit %s", data)
	}
}

func TestInventoryCountsDevicesAndPointsPerNetworkWithLocalApart(t *testing.T) {
	out := ok(t, bqlServer(t, ""), "n4_inventory")
	totals := get(out, "totals").(*pyjson.Object)
	if get(totals, "field_devices") != 4 || get(totals, "local_devices") != 1 || get(totals, "points") != 5 {
		t.Errorf("totals %s", repr(totals))
	}
	var steps, rows []any
	for _, p := range get(out, "progress").([]any) {
		steps, rows = append(steps, get(p.(*pyjson.Object), "step")), append(rows, get(p.(*pyjson.Object), "rows"))
	}
	if repr(steps) != "['networks', 'devices', 'points']" || repr(rows) != "[4, 5, 5]" {
		t.Errorf("progress %s", repr(get(out, "progress")))
	}
	if _, ok := out.Get("point_rows"); ok {
		t.Error("point_rows without include_points")
	}
}

func TestProgressLinesGoToTheOperatorProgressFile(t *testing.T) {
	s := bqlServer(t, "")
	path := filepath.Join(t.TempDir(), "progress.jsonl")
	s.Ctx.ProgressPath = path
	ok(t, s, "n4_inventory", "base", "/Drivers")
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var got []string
	for sc := bufio.NewScanner(f); sc.Scan(); {
		var e struct {
			Tool, Step string
			Rows       int
		}
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
		got = append(got, e.Tool+" "+e.Step+" "+string(rune('0'+e.Rows)))
	}
	if strings.Join(got, ", ") != "n4_inventory networks 4, n4_inventory devices 5, n4_inventory points 5" {
		t.Errorf("progress lines %q", got)
	}
}

func TestIncludePointsReturnsThePointRows(t *testing.T) {
	if rows := get(ok(t, bqlServer(t, ""), "n4_inventory", "include_points", true), "point_rows"); len(rows.([]any)) != 5 {
		t.Errorf("point_rows %s", repr(rows))
	}
}
