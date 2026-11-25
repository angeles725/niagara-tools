package writeops_test

import (
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// capacity_records is an alias of capacity for a number of records (#859: 500 records = 8 h at
// 1 min). It plans exactly what capacity plans; both together are refused.
func TestAddHistoryCapacityRecordsPlansWhatCapacityPlans(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	want, err := planHistory(c, histArgs("capacity", pyjson.IntLiteral("20160")))
	if err != nil {
		t.Fatal(err)
	}
	got, err := planHistory(c, histArgs("capacity_records", pyjson.IntLiteral("20160")))
	if err != nil {
		t.Fatalf("capacity_records was refused: %v", err)
	}
	if js(t, got.Ops) != js(t, want.Ops) || js(t, got.Inverse) != js(t, want.Inverse) {
		t.Errorf("capacity_records ops\n got  %s\n want %s", js(t, got.Ops), js(t, want.Ops))
	}
}

func TestAddHistoryCapacityRecordsAndCapacityTogetherAreRefused(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	for _, tc := range []struct {
		name string
		args *pyjson.Object
	}{
		{"different", histArgs("capacity", pyjson.IntLiteral("100"), "capacity_records", pyjson.IntLiteral("200"))},
		{"equal", histArgs("capacity", pyjson.IntLiteral("100"), "capacity_records", pyjson.IntLiteral("100"))},
		{"unlimited", histArgs("capacity", "unlimited", "capacity_records", pyjson.IntLiteral("100"))},
	} {
		_, err := planHistory(c, tc.args)
		wantRefusal(t, tc.name, err, "capacity and capacity_records are the same argument: give only one of them")
	}
}

func TestAddHistoryCapacityRecordsRefusalsNameTheArgument(t *testing.T) {
	c := histPoints(t, ptA, point("10"), ptB, point("11"))
	for _, tc := range []struct {
		name string
		args *pyjson.Object
		want string
	}{
		{"zero", histArgs("capacity_records", pyjson.IntLiteral("0")),
			"capacity_records must be a whole number of records from 1 to 2147483647, got 0"},
		{"too big", histArgs("capacity_records", pyjson.IntLiteral("2147483648")), "capacity_records must be a whole number"},
		{"text", histArgs("capacity_records", "lots"), "capacity_records must be a whole number of records from 1 to 2147483647, got 'lots'"},
		{"unlimited", histArgs("capacity_records", "unlimited"), "capacity_records must be a whole number of records from 1 to 2147483647, got 'unlimited'"},
	} {
		_, err := planHistory(c, tc.args)
		wantRefusal(t, tc.name, err, tc.want)
	}
}
