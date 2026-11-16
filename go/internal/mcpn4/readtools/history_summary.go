package readtools

import (
	"fmt"
	"math"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The summary of n4_read_history (retro 2026-11-16 M2, "a cuanto llego" in one call): count,
// min and max with their times, first and last record and per-status counts over the whole
// range, from one read of every record with machine-encoded time and value. limit cuts only
// the rows. A range larger than summary_max_rows is summarized from its newest records and
// says so, with the station's whole-range min(value), max(value) beside it.

// Defaults and messages of the summary, as data.
const (
	summaryDefaultRows = 100000
	msgSummaryFailed   = "the summary read (%s) failed: %s"
	msgSummaryEmpty    = "the range holds no records"
	msgSummaryText     = "values are not numeric (e.g. %s): min and max are null"
	msgSummaryPartial  = "computed from the newest %d of %s records (summary_max_rows); range_min and " +
		"range_max are the station's select min(value), max(value) over the whole range, as it renders them"
	msgSummaryAggFail = "; the whole-range select min(value), max(value) failed: %s"
)

// historySummary is the summary of the range, or nil and a note saying why there is none.
// total is the range's record count (nil when unknown).
func (t *Tools) historySummary(id, params string, maxRows, timeout int, total any) (any, string, int) {
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistorySummaryQuery)
	if err != nil {
		return nil, err.Error(), 0
	}
	columns, rows, capped, elapsed, err := t.fetchHistory(ord, maxRows, timeout)
	if err == nil && len(rows) > 0 && len(columns) < 3 {
		err = fmt.Errorf("the answer has %d column(s), expected 3", len(columns))
	}
	if err != nil {
		return nil, fmt.Sprintf(msgSummaryFailed, bql.HistorySummaryQuery, err.Error()), elapsed
	}
	sum := summarize(columns, rows)
	count := total
	if !capped || count == nil {
		count = len(rows)
	}
	out := obj("count", count, "complete", !capped, "records_scanned", len(rows))
	for _, k := range sum.Keys() {
		v, _ := sum.Get(k)
		out.Set(k, v)
	}
	if capped {
		of := "?"
		if total != nil {
			of = pyjson.Repr(total)
		}
		note := fmt.Sprintf(msgSummaryPartial, len(rows), of)
		if prior, ok := out.Get("note"); ok {
			note = pyjson.Str(prior) + "; " + note
		}
		ms := t.wholeRangeMinMax(id, params, timeout, out, &note)
		elapsed += ms
		out.Set("note", note)
	}
	return out, "", elapsed
}

// summarize is numeric, min, max, first, last and status_counts of rows (newest first: the
// first row is the last record), with a note when there is nothing numeric to compare. A tie
// keeps the earliest record: when the value was first reached.
func summarize(columns []string, rows []*pyjson.Object) *pyjson.Object {
	statuses := pyjson.NewObject()
	var minRec, maxRec, firstRec, lastRec *pyjson.Object
	var minVal, maxVal float64
	numeric, nonNumeric := len(rows) > 0, ""
	for _, row := range rows {
		ts, raw, status := cell(row, columns[0]), cell(row, columns[1]), cell(row, columns[2])
		n, _ := statuses.Get(status)
		prev, _ := n.(int)
		statuses.Set(status, prev+1)
		value, isNum := pyjson.ParseFloat(raw)
		var shown any = raw
		if isNum && !math.IsNaN(value) && !math.IsInf(value, 0) {
			shown = value
		} else if !isNum && numeric {
			numeric, nonNumeric = false, raw
		}
		// A nan or inf record (a faulted source) is counted but never compared.
		isNum = isNum && !math.IsNaN(value) && !math.IsInf(value, 0)
		rec := obj("value", shown, "timestamp", ts, "status", status)
		if lastRec == nil {
			lastRec = rec
		}
		firstRec = rec
		if isNum && (minRec == nil || value <= minVal) {
			minRec, minVal = rec, value
		}
		if isNum && (maxRec == nil || value >= maxVal) {
			maxRec, maxVal = rec, value
		}
	}
	out := obj("numeric", numeric, "min", nil, "max", nil, "first", nilIfNone(firstRec), "last", nilIfNone(lastRec),
		"status_counts", statuses)
	if numeric && minRec != nil { // nil when every record is nan or inf
		minTs, _ := minRec.Get("timestamp")
		maxTs, _ := maxRec.Get("timestamp")
		out.Set("min", obj("value", minVal, "timestamp", minTs))
		out.Set("max", obj("value", maxVal, "timestamp", maxTs))
	}
	switch {
	case len(rows) == 0:
		out.Set("note", msgSummaryEmpty)
	case !numeric:
		out.Set("note", fmt.Sprintf(msgSummaryText, pyjson.StrRepr(nonNumeric)))
	}
	return out
}

// wholeRangeMinMax sets range_min and range_max from the station's aggregate, or appends why it
// failed to note; it returns the latency.
func (t *Tools) wholeRangeMinMax(id, params string, timeout int, out *pyjson.Object, note *string) int {
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistoryAggregateQuery)
	if err != nil {
		*note += fmt.Sprintf(msgSummaryAggFail, err.Error())
		return 0
	}
	columns, rows, _, elapsed, err := t.fetchHistory(ord, 1, timeout)
	if err == nil && (len(rows) != 1 || len(columns) < 2) {
		err = fmt.Errorf("the answer has %d row(s) and %d column(s), expected 1 and 2", len(rows), len(columns))
	}
	if err != nil {
		*note += fmt.Sprintf(msgSummaryAggFail, err.Error())
		return elapsed
	}
	out.Set("range_min", cell(rows[0], columns[0]))
	out.Set("range_max", cell(rows[0], columns[1]))
	return elapsed
}

// cell is a row's text at column ("" when absent).
func cell(row *pyjson.Object, column string) string {
	v, _ := row.Get(column)
	return pyjson.Str(v)
}

// nilIfNone is o, or a nil interface (JSON null) when o is nil.
func nilIfNone(o *pyjson.Object) any {
	if o == nil {
		return nil
	}
	return o
}
