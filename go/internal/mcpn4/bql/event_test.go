package bql_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

func TestRecordSourcePathTakesOnlyOneSlotPath(t *testing.T) {
	for source, want := range map[string]string{
		"local:|station:|slot:/Plant/Comp1/AmpsAlarm": "/Plant/Comp1/AmpsAlarm",
		"station:|slot:/Plant/a$20b":                  "/Plant/a$20b",
		"slot:/Plant":                                 "/Plant",
		"slot:/":                                      "",
		"local:|h:/x":                                 "",
		"ip:host|fox:|station:|slot:/Plant":           "",
		"slot:/Plant/a b":                             "",
		"slot:/Plant//x":                              "",
		"slot:/Plant|bql:select *":                    "",
	} {
		got, ok := bql.RecordSourcePath(source)
		if got != want || ok != (want != "") {
			t.Errorf("RecordSourcePath(%q) = %q, %v; want %q", source, got, ok, want)
		}
	}
}
