package bql

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func row(kv ...string) *pyjson.Object {
	o := pyjson.NewObject()
	for i := 0; i < len(kv); i += 2 {
		o.Set(kv[i], kv[i+1])
	}
	return o
}

func TestFromTypeIsTheQueriedSpec(t *testing.T) {
	for q, want := range map[string]string{
		"select name, out from control:NumericWritable where out.value > 1": "control:NumericWritable",
		"SELECT name\nFROM driver:Device":                                   "driver:Device",
		"select timestamp, value":                                           "",
	} {
		if got := FromType(q); got != want {
			t.Errorf("FromType(%q) = %q, want %q", q, got, want)
		}
	}
}

func TestNullColumnsAreTheProjectedNamesNullOnEveryRow(t *testing.T) {
	q := "select slotPath, name, maxOverrideDuration, out.value, fallback from control:NumericWritable"
	cols := []string{"Slot Path", "Name", "M", "Out", "Fallback"}
	rows := []*pyjson.Object{
		row("Slot Path", "NULL", "Name", "NULL", "M", "NULL", "Out", "null", "Fallback", "1.0"),
		row("Slot Path", "NULL", "Name", "NULL", "M", "NULL", "Out", "NULL", "Fallback", "NULL"),
	}
	if got := NullColumns(q, cols, rows); !reflect.DeepEqual(got, []string{"maxOverrideDuration", "out"}) {
		t.Errorf("NullColumns = %q", got)
	}
	if got := NullColumns(q, cols, nil); got != nil {
		t.Errorf("no rows: %q", got)
	}
	if got := NullColumns("select * from control:NumericWritable", cols, rows); got != nil {
		t.Errorf("select *: %q", got)
	}
}

// A TOP or DISTINCT select names the same slots as without the modifier: the survey bug named
// "top 10 slotPath" and, being no pseudo column, warned about a slot that was never queried.
func TestNullColumnsIgnoreTheTopAndDistinctModifiers(t *testing.T) {
	cols := []string{"Slot Path", "Name", "M"}
	rows := []*pyjson.Object{row("Slot Path", "NULL", "Name", "NULL", "M", "NULL")}
	for _, q := range []string{"select top 10 slotPath, name, maxOverrideDuration from control:NumericWritable",
		"select distinct slotPath, name, maxOverrideDuration from control:NumericWritable"} {
		if got := NullColumns(q, cols, rows); !reflect.DeepEqual(got, []string{"maxOverrideDuration"}) {
			t.Errorf("NullColumns(%q) = %q", q, got)
		}
	}
}
