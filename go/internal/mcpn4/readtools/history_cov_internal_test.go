package readtools

import (
	"reflect"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestNearestRankPercentile(t *testing.T) {
	sorted := []float64{0.1, 0.2, 0.2, 0.3, 0.3}
	for p, want := range map[float64]float64{0.5: 0.2, 0.9: 0.3, 0.99: 0.3, 0.2: 0.1} {
		if got := percentile(sorted, p); got != want {
			t.Errorf("percentile(%v) = %v, want %v", p, got, want)
		}
	}
}

func TestRoundUp125IsTheSmallestSeriesValueAtOrAbove(t *testing.T) {
	for x, want := range map[float64]float64{0.1: 0.1, 0.11: 0.2, 0.2: 0.2, 0.3: 0.5, 0.7: 1, 1: 1, 3: 5, 12: 20, 0.004: 0.005} {
		if got := roundUp125(x); got != want {
			t.Errorf("roundUp125(%v) = %v, want %v", x, got, want)
		}
	}
	for x, want := range map[float64]float64{0.1: 0.2, 0.2: 0.5, 0.5: 1, 5: 10} {
		if got := next125(x); got != want {
			t.Errorf("next125(%v) = %v, want %v", x, got, want)
		}
	}
}

func TestSimulatedCovKeepsARecordOnlyBeyondTheTolerance(t *testing.T) {
	values := []float64{20.0, 20.3, 20.1, 20.2, 20.0, 20.3}
	for tol, want := range map[float64]int{0.2: 4, 0.3: 1, 0.5: 1, 0.05: 6} {
		if got := covKept(values, tol); got != want {
			t.Errorf("covKept(%v) = %d, want %d", tol, got, want)
		}
	}
}

func TestCovSuggestWalksTheSeriesUntilHalfTheRecordsGo(t *testing.T) {
	got := covSuggest([]float64{20.0, 20.3, 20.1, 20.2, 20.0, 20.3})
	want := "{'suggested_tolerance': 0.5, 'records_considered': 6, " +
		"'deltas': {'count': 5, 'median': 0.2, 'p90': 0.3, 'p99': 0.3, 'nonzero_median': 0.2, 'noise_floor': 0.1, 'zero_share': 0.0}, " +
		"'candidates': [{'tolerance': 0.2, 'deltas_within': 0.6, 'reduction': 0.3333333333}, " +
		"{'tolerance': 0.5, 'deltas_within': 1.0, 'reduction': 0.8333333333}], 'reduction_estimate': 0.8333333333}"
	if g := pyjson.Repr(got); g != want {
		t.Errorf("covSuggest =\n%s\nwant\n%s", g, want)
	}
}

func TestCovSuggestOfAConstantValueSuggestsNothing(t *testing.T) {
	got := covSuggest([]float64{5, 5, 5})
	if v, _ := got.Get("suggested_tolerance"); v != nil {
		t.Errorf("suggested_tolerance = %v, want null", v)
	}
	if v, _ := got.Get("note"); v != msgCovConstant {
		t.Errorf("note = %v", v)
	}
}

func TestCovValuesKeepsFiniteNonNullRecordsOldestFirst(t *testing.T) {
	cols := []string{"t", "v", "s"}
	row := func(v, s string) *pyjson.Object { return obj("t", "x", "v", v, "s", s) }
	rows := []*pyjson.Object{row("3.0", "{ok}"), row("nan", "{fault}"), row("0.0", "{null}"), row("1.5", "{stale}")}
	values, bad := covValues(cols, rows)
	if !reflect.DeepEqual(values, []float64{1.5, 3.0}) || bad != "" {
		t.Errorf("values %v bad %q", values, bad)
	}
	if _, bad = covValues(cols, []*pyjson.Object{row("true", "{ok}")}); bad != "true" {
		t.Errorf("non-numeric bad = %q", bad)
	}
}
