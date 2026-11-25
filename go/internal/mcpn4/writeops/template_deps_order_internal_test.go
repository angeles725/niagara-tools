package writeops

import (
	"sort"
	"testing"
)

// tplVersionLess is a total order (#1070): numeric prefix per component, then a release above a
// suffixed one (1.2 above 1.2-SNAPSHOT), then the suffix text; a missing component is 0 and one with no number sorts first.
func TestTplVersionLessIsATotalOrder(t *testing.T) {
	sorted := []string{"x", "1.2-SNAPSHOT", "1.2", "1.2.0.1", "1.10-rc1", "1.10", "4.14.0.162", "4.14.1"}
	got := append([]string(nil), sorted...)
	for i := 0; i < 20; i++ { // several shuffles, one fixed answer
		got[0], got[len(got)-1-i%len(got)] = got[len(got)-1-i%len(got)], got[0]
		sort.Slice(got, func(i, j int) bool { return tplVersionLess(got[i], got[j]) })
		for k := range sorted {
			if got[k] != sorted[k] {
				t.Fatalf("sorted %v, want %v", got, sorted)
			}
		}
	}
	for _, v := range sorted {
		if tplVersionLess(v, v) {
			t.Errorf("%s < %s", v, v)
		}
	}
	if tplVersionLess("1.2", "1.2.0") || tplVersionLess("1.2.0", "1.2") {
		t.Error("1.2 and 1.2.0 are equal")
	}
}

// Several loaded parts of one module: the highest version is compared, whatever the map order.
func TestTplCompareDepsTakesTheHighestPartVersion(t *testing.T) {
	for i := 0; i < 50; i++ {
		parts := map[string]string{"acme-rt": "1.2-SNAPSHOT", "acme-ux": "1.2", "acme-wb": "1.1"}
		_, older, met := tplCompareDeps([]string{"acme 1.2"}, parts)
		if met != 1 || len(older) != 0 {
			t.Fatalf("run %d: older %v met %d", i, older, met)
		}
	}
}
