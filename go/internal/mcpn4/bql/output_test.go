package bql_test

// Go twins of test_bql.TestLocalizedHeaders, TestBqlOutputFile and TestH5aRegressions
// (go_server.go_twin("bql", "injection")), and the result-file permission rules the plan pins:
// dirs 0700, file 0600, O_EXCL, an existing file or directory never chmodded.

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest"
)

// localized is a fixture with Spanish display headers (the station localizes them).
func localized(text string) string {
	head, rest, _ := strings.Cut(text, "\n")
	head = strings.NewReplacer("Slot Path", "Ruta de slot", "Name", "Nombre", "Type", "Tipo", "Out", "Salida").Replace(head)
	return head + "\n" + rest
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func TestInventoryDoesNotDependOnEnglishHeaders(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = routes(t, localized)
	out := ok(t, s, "n4_inventory")
	if got := repr(get(out, "totals")); got != "{'networks': 4, 'field_devices': 4, 'local_devices': 1, 'points': 5, 'unassigned_points': 0}" {
		t.Errorf("totals %s", got)
	}
	if !strings.Contains(repr(get(out, "devices")), "'name': 'Meter A'") {
		t.Errorf("devices %s", repr(get(out, "devices")))
	}
}

func TestBqlQueryDecodesTheQueriedNameColumn(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes = routes(t, localized)
	out := ok(t, s, "n4_bql_query", "base", "/Drivers", "query", devicesQuery)
	rows := repr(get(out, "rows"))
	if repr(get(out, "columns")) != "['Ruta de slot', 'Nombre', 'Tipo']" || !strings.Contains(rows, "'Nombre': 'Meter A'") ||
		!strings.Contains(rows, "'Ruta de slot': 'slot:/Drivers/ModbusTcpNetwork/Meter$20A'") {
		t.Errorf("reply %s", repr(out))
	}
}

func TestRowsGoToAPrivateFileUnderTheStateDirNotIntoTheReply(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	s := bqlServer(t, state)
	out := ok(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "devices.json", "query", devicesQuery)
	path := filepath.Join(state, "bql", "devices.json")
	if _, inReply := out.Get("rows"); get(out, "output_file") != path || inReply || get(out, "row_count") != 5 {
		t.Errorf("reply %s", repr(out))
	}
	if mode(t, path) != 0o600 || mode(t, filepath.Dir(path)) != 0o700 || mode(t, state) != 0o700 {
		t.Errorf("modes file %o, bql %o, state %o", mode(t, path), mode(t, filepath.Dir(path)), mode(t, state))
	}
	data, _ := os.ReadFile(path)
	var saved struct {
		Columns []string
		Rows    []map[string]string
	}
	if err := json.Unmarshal(data, &saved); err != nil || strings.Join(saved.Columns, ",") != "Slot Path,Name,Type" || len(saved.Rows) != 5 {
		t.Errorf("saved %s (%v)", data, err)
	}
}

func TestACsvOutputFileKeepsTheColumnsInOrder(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	ok(t, bqlServer(t, state), "n4_bql_query", "base", "/Drivers", "output_file", "devices.csv", "query", devicesQuery)
	f, err := os.Open(filepath.Join(state, "bql", "devices.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	lines, err := csv.NewReader(f).ReadAll()
	if err != nil || strings.Join(lines[0], ",") != "Slot Path,Name,Type" || len(lines) != 6 {
		t.Errorf("csv %q (%v)", lines, err)
	}
}

func TestAPathOrAnExistingFileIsRefusedAndNothingIsRead(t *testing.T) {
	s := bqlServer(t, filepath.Join(t.TempDir(), "state"))
	for _, bad := range []string{"../x.json", "a/b.json", "/tmp/x.json", "x.txt", ".hidden.json", ""} {
		has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", bad, "query", "select name from driver:Device"), "output_file")
	}
	if len(s.Station.OrdRequests) != 0 {
		t.Errorf("requests sent: %+v", s.Station.OrdRequests)
	}
	ok(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "d.json", "query", "select name from driver:Device")
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "d.json", "query", "select name from driver:Device"), "exists")
}

func TestAHeaderOnlyBodyCountsZeroRowsInsteadOfFailingTheInventory(t *testing.T) {
	s := bqlServer(t, "")
	s.Station.OrdRoutes[1].Body = "\ufeffSlot Path,Name,Type\n"
	s.Station.OrdRoutes[2].Body = "Slot Path,Name,Type,Out\n"
	totals := get(ok(t, s, "n4_inventory"), "totals").(*pyjson.Object)
	if get(totals, "field_devices") != 0 || get(totals, "points") != 0 {
		t.Errorf("totals %s", repr(totals))
	}
}

// The Python test fails json.dump mid-write; here the write fails twice without a seam: the
// result dir is unwritable (open fails), then a cell holds a lone surrogate (the file is created,
// its row cannot be encoded, and it is removed).
func TestAFailedOutputWriteLeavesNoPartialFileAndTheNameStaysUsable(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	s := bqlServer(t, state)
	path := filepath.Join(state, "bql", "d.csv")
	if os.Getuid() != 0 { // root writes through 0500
		os.MkdirAll(filepath.Dir(path), 0o700)
		os.Chmod(filepath.Dir(path), 0o500)
		has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "d.csv", "query", "select name from driver:Device"),
			"could not be written (Permission denied)")
		os.Chmod(filepath.Dir(path), 0o700)
	}
	s.Station.OrdRoutes = []readtoolstest.OrdRoute{{Needle: "driver:Device", Body: "Name\nok\nA$ud800B\n"}}
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "d.csv", "query", "select name from driver:Device"),
		`'utf-8' codec can't encode character '\ud800' in position 1: surrogates not allowed`)
	if _, err := os.Lstat(path); err == nil {
		t.Error("a partial result file was left behind")
	}
	s.Station.OrdRoutes = routes(t, nil)
	ok(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "d.csv", "query", "select name from driver:Device")
	if _, err := os.Lstat(path); err != nil {
		t.Error(err)
	}
}

func TestAnExistingResultDirKeepsItsModeAndALinkIsNeverFollowed(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	os.MkdirAll(filepath.Join(state, "bql"), 0o700)
	os.Chmod(filepath.Join(state, "bql"), 0o750)
	s := bqlServer(t, state)
	ok(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "a.json", "query", devicesQuery)
	if m := mode(t, filepath.Join(state, "bql")); m != 0o750 {
		t.Errorf("an existing result dir was chmodded to %o", m)
	}
	target := filepath.Join(t.TempDir(), "elsewhere")
	os.Symlink(target, filepath.Join(state, "bql", "b.json")) // dangling: lexists, never written through
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "b.json", "query", devicesQuery), "already exists")
	if _, err := os.Lstat(target); err == nil {
		t.Error("the link target was created")
	}
}

func TestAnUntrustedStateDirIsRefusedAndNeverChmodded(t *testing.T) {
	state := t.TempDir()
	os.Chmod(state, 0o777)
	s := bqlServer(t, state)
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "a.json", "query", devicesQuery),
		"--state-dir "+state+" is group/world-writable (mode 777)")
	if mode(t, state) != 0o777 || len(s.Station.OrdRequests) != 0 {
		t.Errorf("mode %o, %d requests", mode(t, state), len(s.Station.OrdRequests))
	}
	os.Chmod(state, 0o700)
	os.WriteFile(filepath.Join(state, "audit.jsonl"), nil, 0o644)
	os.Chmod(filepath.Join(state, "audit.jsonl"), 0o644)
	has(t, fail(t, s, "n4_bql_query", "base", "/Drivers", "output_file", "a.json", "query", devicesQuery),
		"audit.jsonl is readable or writable by group/others (mode 644)")
}
