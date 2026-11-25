package bql_test

// Go twins of the query and ORD tests of test_bql.TestPureHelpers and TestHeaderRobustness
// (go_server.go_twin("bql", "unit")).

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

func TestUnescapeSlotDecodesDollarEscapes(t *testing.T) {
	for in, want := range map[string]string{"Meter$20Side$20A": "Meter Side A", "Ahu$2dUnit$2001_1": "Ahu-Unit 01_1",
		"Caf$u00e9": "Café", "plain": "plain"} {
		if got := bql.UnescapeSlot(in); got != want {
			t.Errorf("UnescapeSlot(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateQueryAcceptsSelectOnly(t *testing.T) {
	where := "select name, out from control:ControlPoint where name like 'T*'"
	for in, want := range map[string]string{"  SELECT name from driver:Device ": "SELECT name from driver:Device", where: where} {
		if got, err := bql.ValidateQuery(in); err != nil || got != want {
			t.Errorf("ValidateQuery(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []any{"delete from control:ControlPoint", "name from x", "", "select", "selectname from x", 5, nil} {
		if _, err := bql.ValidateQuery(bad); err == nil {
			t.Errorf("ValidateQuery(%#v) accepted", bad)
		}
	}
}

func TestValidateQueryRefusesPipeAndControlCharacters(t *testing.T) {
	for _, bad := range []string{"select name from driver:Device|view:file:ITableToCsv",
		"select name from control:ControlPoint where name = 'a|b'", "select name\nfrom driver:Device", "select name from x\x00"} {
		if _, err := bql.ValidateQuery(bad); err == nil {
			t.Errorf("ValidateQuery(%q) accepted", bad)
		}
	}
}

func TestComposeOrdBuildsTheOneProjectedOrd(t *testing.T) {
	q := "select name from driver:Device"
	for _, base := range []string{"station:|slot:/Drivers", "/Drivers", "slot:/Drivers", "station:|slot:/Drivers/"} {
		if got, err := bql.ComposeOrd(base, q); err != nil || got != "station:|slot:/Drivers|bql:"+q+"|view:file:ITableToCsv" {
			t.Errorf("ComposeOrd(%q) = %q, %v", base, got, err)
		}
	}
	if got, _ := bql.ComposeOrd("station:|slot:/", q); got != "station:|slot:/|bql:"+q+"|view:file:ITableToCsv" {
		t.Errorf("root: %q", got)
	}
}

func TestComposeOrdRefusesABaseThatInjectsAnotherScheme(t *testing.T) {
	for _, base := range []string{"station:|slot:/Drivers|bql:select x from y", "station:|h:12", "file:^x",
		"station:|slot://Drivers", "station:|slot:/A\nB", ""} {
		if _, err := bql.ComposeOrd(base, "select name from driver:Device"); err == nil {
			t.Errorf("ComposeOrd(%q) accepted", base)
		}
	}
}

func TestCompleteRecordsCutsOnlyOutsideQuotes(t *testing.T) {
	for in, want := range map[string]string{"A,B\nz,\"ok\"\nx,\"line one\nli": "A,B\nz,\"ok\"\n",
		"A\n\"say \"\"hi\"\"\nthere\"\n": "A\n\"say \"\"hi\"\"\nthere\"\n", "A\r\n\"x": "A\r\n", "A,B": ""} {
		if got := bql.CompleteRecords(in); got != want {
			t.Errorf("CompleteRecords(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSelectedSlotsNameTheQueriedColumnsInOrder(t *testing.T) {
	if got := bql.SelectedSlots("select slotPath, name ,type from driver:Device"); !reflect.DeepEqual(got, []string{"slotPath", "name", "type"}) {
		t.Errorf("%q", got)
	}
	if got := bql.SelectedSlots("SELECT proxyExt.dataAddress FROM x where a = 1"); !reflect.DeepEqual(got, []string{"proxyExt.dataAddress"}) {
		t.Errorf("%q", got)
	}
	if got := bql.SelectedSlots("select * from driver:Device"); got != nil {
		t.Errorf("*: %q", got)
	}
}

// Field retro 2026-11-10 row 7: a query written as station:|slot:/Drivers/...|bql:select ... was
// refused with a bare "query must not contain '|'". The refusal says where the subtree goes.
func TestThePipeRefusalNamesTheBaseAsTheFix(t *testing.T) {
	_, err := bql.ValidateQuery("station:|slot:/Drivers/Net|bql:select name from control:ControlPoint")
	want := "query must not contain '|': only one select is composed into the ORD; to query a subtree put it in " +
		"base (for example base station:|slot:/Drivers/Net) and keep only 'select ... from ...' in query"
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	// A query with no '|' is unchanged.
	if _, err := bql.ValidateQuery("select name from control:ControlPoint"); err != nil {
		t.Errorf("a plain select refused: %v", err)
	}
}

// BQL reads select [TOP n] [DISTINCT|ALL] <columns> (bql-rt SelectParser.parse: top(), then
// quantifier(), then projection()); the modifiers are no column. API survey 2026-11-24: the first
// name of "select top 10 slotPath, name" was "top 10 slotPath".
func TestSelectedSlotsDropTheTopAndQuantifierModifiers(t *testing.T) {
	for q, want := range map[string][]string{
		"select top 10 slotPath, name from baja:Component":   {"slotPath", "name"},
		"SELECT TOP 5 name FROM baja:Component":              {"name"},
		"select  Top\t7\nslotPath , name from x where a = 1": {"slotPath", "name"},
		"select distinct type from baja:Component":           {"type"},
		"select DISTINCT a, b from x":                        {"a", "b"},
		"select all a, b from x":                             {"a", "b"},
		"select top 3 distinct a, b from x":                  {"a", "b"},
		"select TOP 3 ALL a from x":                          {"a"},
		"select top 10 * from x":                             nil,
		"select topSpeed, distinctCount, allowed from x":     {"topSpeed", "distinctCount", "allowed"},
		"select top, distinct from x":                        {"top", "distinct"},
		"select top from x":                                  {"top"},
		"select top 10, name from x":                         {"top 10", "name"},
	} {
		if got := bql.SelectedSlots(q); !reflect.DeepEqual(got, want) {
			t.Errorf("SelectedSlots(%q) = %q, want %q", q, got, want)
		}
	}
}

// A TOP or DISTINCT select is a plain select to the query check, so it reaches SelectedSlots.
func TestValidateQueryAcceptsTopAndDistinct(t *testing.T) {
	for _, q := range []string{"select top 10 slotPath, name from baja:Component", "select distinct type from baja:Component"} {
		if got, err := bql.ValidateQuery(q); err != nil || got != q {
			t.Errorf("ValidateQuery(%q) = %q, %v", q, got, err)
		}
	}
}
