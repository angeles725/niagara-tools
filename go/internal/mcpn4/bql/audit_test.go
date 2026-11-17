package bql_test

import (
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
)

func TestAuditWhereBuildsEveryFilter(t *testing.T) {
	for _, c := range []struct {
		f    bql.AuditFilter
		want string
	}{
		{bql.AuditFilter{}, ""},
		{bql.AuditFilter{Slot: "overAmpsDelay"}, "slotName = 'overAmpsDelay'"},
		{bql.AuditFilter{Slot: "facets/units"}, "slotName = 'facets/units'"},
		{bql.AuditFilter{User: "admin"}, "userName = 'admin'"},
		{bql.AuditFilter{Operation: "Flags Changed"}, "operation = 'Flags Changed'"},
		{bql.AuditFilter{Target: "station:|slot:/Plant/Comp1"},
			"(target like '/Plant/Comp1' or target like '/Plant/Comp1/*')"},
		{bql.AuditFilter{Target: "slot:/Plant/a$2420b"},
			"(target like '/Plant/a$2420b' or target like '/Plant/a$2420b/*')"},
		{bql.AuditFilter{TargetGlob: "/Plant/*/Amps*"}, "target like '/Plant/*/Amps*'"},
		{bql.AuditFilter{Target: "slot:/Plant", Slot: "overAmpsDelay", User: "admin", Operation: "Changed"},
			"(target like '/Plant' or target like '/Plant/*') and slotName = 'overAmpsDelay' and " +
				"userName = 'admin' and operation = 'Changed'"},
	} {
		got, err := bql.AuditWhere(c.f)
		if err != nil || got != c.want {
			t.Errorf("AuditWhere(%+v) = %q, %v; want %q", c.f, got, err, c.want)
		}
	}
}

func TestAuditWhereRefusesInjectionShapedValuesOnTheirDecodedForm(t *testing.T) {
	for _, c := range []struct {
		f    bql.AuditFilter
		want string
	}{
		{bql.AuditFilter{Slot: "x' or slotName like '*"}, "slot must be"},
		{bql.AuditFilter{Slot: "x$27$20or$20slotName$20like$20$27$2a"}, "BQL decodes to"},
		{bql.AuditFilter{Slot: "x$2"}, "slot must be"},
		{bql.AuditFilter{User: "admin' or userName like '*"}, "user must be"},
		{bql.AuditFilter{User: "adm$27in"}, "user must be"},
		{bql.AuditFilter{Operation: "Changed' or operation like '*"}, "operation must be one of"},
		{bql.AuditFilter{Target: "station:|slot:/Plant/x$27"}, "BQL decodes to"},
		{bql.AuditFilter{Target: "slot:/Plant|bql:select *"}, "target must be"},
		{bql.AuditFilter{Target: "/Plant"}, "target must be"},
		{bql.AuditFilter{TargetGlob: "/Plant/*' or target like '*"}, "target_glob must be"},
		{bql.AuditFilter{TargetGlob: "/Plant/$27*"}, "target_glob must be"},
		{bql.AuditFilter{Target: "slot:/Plant", TargetGlob: "/Plant/*"}, "give at most one of target"},
	} {
		got, err := bql.AuditWhere(c.f)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("AuditWhere(%+v) = %q, %v; want a refusal holding %q", c.f, got, err, c.want)
		}
	}
}

func TestComposeFilteredHistoryOrdReadsTheAuditHistoryNewestFirst(t *testing.T) {
	got, err := bql.ComposeFilteredHistoryOrd("/Lab/AuditHistory", "period=today", bql.AuditRecordsQuery, "slotName = 'x'", bql.AuditOrder)
	want := "history:/Lab/AuditHistory?period=today|bql:select timestamp.encodeToString, operation, target, " +
		"slotName, oldValue, value, userName where slotName = 'x' order by timestamp desc|view:file:ITableToCsv"
	if err != nil || got != want {
		t.Errorf("records ORD %q, %v\nwant %q", got, err, want)
	}
	got, err = bql.ComposeFilteredHistoryOrd("/Lab/AuditHistory", "", bql.HistoryCountQuery, "", "")
	if want := "history:/Lab/AuditHistory|bql:select count(*)|view:file:ITableToCsv"; err != nil || got != want {
		t.Errorf("count ORD %q, %v; want %q", got, err, want)
	}
}
