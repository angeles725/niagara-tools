package bql_test

// Go twins of the CSV tests of test_bql.TestPureHelpers and TestHeaderRobustness
// (go_server.go_twin("bql", "unit")).

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// fixture is a recorded station answer of mcp-n4-kit/tests/fixtures.
func fixture(t testing.TB, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func parse(t testing.TB, text string, max int, names []int) ([]string, []*pyjson.Object, bool) {
	t.Helper()
	cols, rows, cut, err := bql.ParseCSV(text, max, names)
	if err != nil {
		t.Fatalf("ParseCSV(%q): %v", text, err)
	}
	return cols, rows, cut
}

// reprs is the rows as Python would print them, for one comparison.
func reprs(rows []*pyjson.Object) string {
	list := make([]any, len(rows))
	for i, r := range rows {
		list[i] = r
	}
	return pyjson.Repr(list)
}

func get(o *pyjson.Object, k string) any { v, _ := o.Get(k); return v }

func TestParseCsvStripsBomAndControlCharsAndCapsRows(t *testing.T) {
	text := fixture(t, "bql_points.csv")
	cols, rows, cut := parse(t, text, 100, nil)
	if !reflect.DeepEqual(cols, []string{"Slot Path", "Name", "Type", "Out", "Data Address"}) || len(rows) != 5 || cut {
		t.Fatalf("columns %q, %d rows, truncated %v", cols, len(rows), cut)
	}
	if get(rows[1], "Out") != "1,024.0 {ok}" || get(rows[0], "Data Address") != "Decimal:302" {
		t.Errorf("cells: %s", reprs(rows[:2]))
	}
	if _, rows, cut = parse(t, text, 2, nil); len(rows) != 2 || !cut {
		t.Errorf("cap 2: %d rows, truncated %v", len(rows), cut)
	}
	if _, rows, _ = parse(t, "A,B\r\nx\x07y,\x1bz\r\n", 5, nil); reprs(rows) != "[{'A': 'xy', 'B': 'z'}]" {
		t.Errorf("control chars: %s", reprs(rows))
	}
}

func TestParseCsvDecodesTheNameColumnAndKeepsTheOrdReadyPath(t *testing.T) {
	_, rows, _ := parse(t, fixture(t, "bql_devices.csv"), 10, nil)
	if get(rows[0], "Name") != "Meter A" || get(rows[0], "Slot Path") != "slot:/Drivers/ModbusTcpNetwork/Meter$20A" {
		t.Errorf("row 0: %s", reprs(rows[:1]))
	}
}

func TestParseCsvRefusesABodyWithoutAHeaderRow(t *testing.T) {
	for _, text := range []string{"", "\ufeff", "\r\n"} {
		if _, _, _, err := bql.ParseCSV(text, 5, nil); err == nil || !strings.Contains(err.Error(), "no header row") {
			t.Errorf("ParseCSV(%q): %v", text, err)
		}
	}
	if cols, rows, cut := parse(t, "\ufeffA,B\n", 5, nil); !reflect.DeepEqual(cols, []string{"A", "B"}) || len(rows) != 0 || cut {
		t.Errorf("header only: %q %d %v", cols, len(rows), cut)
	}
}

func TestParseCsvRefusesARaggedRecord(t *testing.T) {
	for text, n := range map[string]string{"A,B\nok,ok\nz\n": "1", "A,B\nok,ok\nx,y,extra\n": "3"} {
		if _, _, _, err := bql.ParseCSV(text, 5, nil); err == nil || !strings.Contains(err.Error(), "record 2 has "+n+" cell") {
			t.Errorf("ParseCSV(%q): %v", text, err)
		}
	}
}

func TestDuplicateHeadersKeepEveryColumn(t *testing.T) {
	cols, rows, _ := parse(t, "Type,Type,Type#2,Type\nA,B,C,D\n", bql.DefaultMaxRows, nil)
	if !reflect.DeepEqual(cols, []string{"Type", "Type#2", "Type#2#2", "Type#3"}) ||
		reprs(rows) != "[{'Type': 'A', 'Type#2': 'B', 'Type#2#2': 'C', 'Type#3': 'D'}]" {
		t.Errorf("%q %s", cols, reprs(rows))
	}
}

func TestTheNameColumnIsDecodedByPositionWhateverItsHeader(t *testing.T) {
	if _, rows, _ := parse(t, "Ruta,Nombre\nslot:/A$20B,A$20B\n", bql.DefaultMaxRows, []int{1}); reprs(rows) != "[{'Ruta': 'slot:/A$20B', 'Nombre': 'A B'}]" {
		t.Errorf("%s", reprs(rows))
	}
}

func TestByPositionMapsRowsToTheQueriedSlotNames(t *testing.T) {
	columns := []string{"Ruta", "Nombre", "Tipo"}
	_, rows, _ := parse(t, "Ruta,Nombre,Tipo\nslot:/D/N,N,x:Net\n", 5, nil)
	names := []string{"Slot Path", "Name", "Type"}
	if got, err := bql.ByPosition(columns, rows, names); err != nil || reprs(got) != "[{'Slot Path': 'slot:/D/N', 'Name': 'N', 'Type': 'x:Net'}]" {
		t.Errorf("%s %v", reprs(got), err)
	}
	if _, err := bql.ByPosition(columns[:2], rows, names); err == nil || !strings.Contains(err.Error(), "2 column(s), expected at least 3") {
		t.Errorf("short: %v", err)
	}
}
