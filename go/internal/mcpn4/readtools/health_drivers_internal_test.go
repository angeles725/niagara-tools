package readtools

import (
	"reflect"
	"testing"
)

func TestDriverCategoriesCountsEachBitCategoryOnce(t *testing.T) {
	cases := []struct {
		bits    string
		enabled any
		cats    []string
		want    []string
	}{
		{"0", true, pointCategories, nil},
		{"20", nil, pointCategories, []string{"stale", "down"}}, // down + stale
		{"23", nil, pointCategories, []string{"stale", "fault", "down", "disabled"}},
		{"8", nil, pointCategories, nil},  // alarm is no driver category
		{"64", nil, pointCategories, nil}, // null alone: no value, no fault
		{"", nil, pointCategories, []string{unknownFlag}},
		{"0", false, deviceCategories, []string{"disabled"}}, // enabled false is disabled
		{"1", false, deviceCategories, []string{"disabled"}}, // counted once
		{"16", nil, deviceCategories, nil},                   // stale is no device category
	}
	for _, c := range cases {
		if got := driverCategories(statusFlags(c.bits), c.enabled, c.cats); !reflect.DeepEqual(got, c.want) {
			t.Errorf("bits %q enabled %v: %v, want %v", c.bits, c.enabled, got, c.want)
		}
	}
}

func TestIsProxyPointLeavesOutNullProxyExtAndUnknown(t *testing.T) {
	for ext, want := range map[string]bool{"bacnet:BacnetNumericProxyExt": true, nullProxyExt: false, "": false, " ": false} {
		if got := isProxyPoint(obj(proxyColumn, ext)); got != want {
			t.Errorf("%q: %v", ext, got)
		}
	}
}
