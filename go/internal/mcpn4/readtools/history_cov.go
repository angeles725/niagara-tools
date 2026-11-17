package readtools

import (
	"fmt"
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_history cov_hint (station-audits S3): a COV changeTolerance suggested from the
// history's real data, from the same machine-encoded read the summary uses. Report only: the
// server never writes the tolerance.

// The rule and the notes of cov_hint, as data.
const (
	covRule = "suggested_tolerance is the smallest value of the 1-2-5 series (..., 0.1, 0.2, 0.5, 1, 2, 5, 10, ...) " +
		"at or above deltas.nonzero_median (the nearest-rank median of the non-zero |delta| between successive " +
		"considered records) for which a simulated COV over these records keeps at most half of them " +
		"(reduction >= 0.5). The simulation keeps the first record, then each record whose |value - last kept " +
		"value| is above the tolerance, as BNumericCovHistoryExt.isChange compares (a status change, which " +
		"always records, is not simulated). Considered records: a finite numeric value and a status without the " +
		"null flag, oldest first; deltas are rounded to 10 significant digits."
	msgCovText     = "values are not numeric (e.g. %s): a COV changeTolerance applies to numeric histories only"
	msgCovFew      = "fewer than 2 numeric records with a value in the range: no change to measure"
	msgCovConstant = "the value never changed between considered records: any tolerance keeps one record, so none is suggested"
)

// covHint is the cov_hint of one summary read, or nil and a note saying why there is none.
func covHint(read summaryRead) (any, string) {
	if read.failure != "" {
		return nil, read.failure
	}
	if len(read.rows) == 0 {
		return nil, msgSummaryEmpty
	}
	values, bad := covValues(read.columns, read.rows)
	switch {
	case bad != "":
		return nil, fmt.Sprintf(msgCovText, pyjson.StrRepr(bad))
	case len(values) < 2:
		return nil, msgCovFew
	}
	body := covSuggest(values)
	out := pyjson.NewObject()
	for _, k := range body.Keys() {
		v, _ := body.Get(k)
		out.Set(k, v)
		if k == "suggested_tolerance" {
			out.Set("rule", covRule)
		} else if k == "records_considered" {
			out.Set("complete", !read.capped)
		}
	}
	return out, ""
}

// covValues is the considered values of rows (newest first) oldest first: a finite number whose
// status has no null flag; a record with no value text is skipped. bad is the first value that
// is not a number at all (a boolean or enum history), "" when there is none.
func covValues(columns []string, rows []*pyjson.Object) ([]float64, string) {
	var values []float64
	for i := len(rows) - 1; i >= 0; i-- {
		raw, status := cell(rows[i], columns[1]), cell(rows[i], columns[2])
		if statusIsNull(status) || raw == "" {
			continue
		}
		v, ok := pyjson.ParseFloat(raw)
		if !ok {
			return nil, raw
		}
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			values = append(values, v)
		}
	}
	return values, ""
}
