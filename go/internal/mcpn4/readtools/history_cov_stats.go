package readtools

import (
	"math"
	"sort"
	"strconv"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The arithmetic of n4_read_history cov_hint (station-audits S3): the |delta| distribution of a
// numeric history and the tolerance a simulated COV would need to drop half of its records.

// covMaxSteps bounds the walk up the 1-2-5 series (each step at least doubles the tolerance, so
// a tolerance above the whole value range, which keeps one record, is reached long before).
const covMaxSteps = 64

// sig is x rounded to 10 significant digits, so 20.1 - 20.0 is 0.1 and not 0.10000000000000142.
func sig(x float64) float64 {
	v, _ := strconv.ParseFloat(strconv.FormatFloat(x, 'g', 10, 64), 64)
	return v
}

// percentile is the nearest-rank p-th percentile of sorted (ascending, not empty).
func percentile(sorted []float64, p float64) float64 {
	i := int(math.Ceil(p*float64(len(sorted)))) - 1
	return sorted[max(0, min(i, len(sorted)-1))]
}

// roundUp125 is the smallest value of the 1-2-5 series (..., 0.1, 0.2, 0.5, 1, 2, 5, 10, ...) at
// or above x (x > 0).
func roundUp125(x float64) float64 {
	e := int(math.Floor(math.Log10(x)))
	for d := e - 1; d <= e+1; d++ {
		for _, m := range []float64{1, 2, 5} {
			if c := sig(m * math.Pow10(d)); c >= x {
				return c
			}
		}
	}
	return sig(math.Pow10(e + 2))
}

// next125 is the series value after x (x a series value).
func next125(x float64) float64 { return roundUp125(x * (1 + 1e-9)) }

// covKept is how many of values (oldest first) a COV with tolerance records: the first, then each
// value whose |value - last recorded value| is above tolerance (BNumericCovHistoryExt.isChange).
func covKept(values []float64, tolerance float64) int {
	kept, last := 1, values[0]
	for _, v := range values[1:] {
		if sig(math.Abs(v-last)) > tolerance {
			kept, last = kept+1, v
		}
	}
	return kept
}

// covSuggest is the cov_hint body of values (oldest first, at least two): the |delta| statistics,
// each tolerance tried with its share of deltas within it and its reduction, and the suggestion.
func covSuggest(values []float64) *pyjson.Object {
	n := len(values)
	sorted := make([]float64, 0, n-1)
	for i := 1; i < n; i++ {
		sorted = append(sorted, sig(math.Abs(values[i]-values[i-1])))
	}
	sort.Float64s(sorted)
	firstNonZero := sort.Search(len(sorted), func(i int) bool { return sorted[i] > 0 })
	nonzero := sorted[firstNonZero:]
	deltas := obj("count", len(sorted), "median", percentile(sorted, 0.5), "p90", percentile(sorted, 0.9),
		"p99", percentile(sorted, 0.99), "nonzero_median", nil, "noise_floor", nil,
		"zero_share", sig(float64(firstNonZero)/float64(len(sorted))))
	candidates := []any{}
	out := obj("suggested_tolerance", nil, "records_considered", n, "deltas", deltas)
	if len(nonzero) == 0 {
		out.Set("candidates", candidates)
		out.Set("reduction_estimate", nil)
		out.Set("note", msgCovConstant)
		return out
	}
	median := percentile(nonzero, 0.5)
	deltas.Set("nonzero_median", median)
	deltas.Set("noise_floor", nonzero[0])
	var suggested, reduction any
	for tol, step := roundUp125(median), 0; step < covMaxSteps; tol, step = next125(tol), step+1 {
		within := sort.Search(len(sorted), func(i int) bool { return sorted[i] > tol })
		red := sig(1 - float64(covKept(values, tol))/float64(n))
		candidates = append(candidates, obj("tolerance", tol, "deltas_within", sig(float64(within)/float64(len(sorted))),
			"reduction", red))
		if red >= 0.5 {
			suggested, reduction = tol, red
			break
		}
	}
	out.Set("suggested_tolerance", suggested)
	out.Set("candidates", candidates)
	out.Set("reduction_estimate", reduction)
	return out
}
