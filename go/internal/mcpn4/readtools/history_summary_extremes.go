package readtools

import (
	"fmt"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Records the summary counts but leaves out of min, max, first and last (retro 2026-11-16 M16):
// a record whose status carries the null flag holds no value (javax.baja.status.BStatusValue
// renders it "-"; a new COV history writes 0.0 {null} records at start), and a nan or inf
// record is a faulted source's non-reading. Other flags (fault, down, stale, disabled) keep
// their value, as BStatusValue still renders it, so they stay in.

const (
	statusNullTag  = "null"
	msgSummaryExcl = "%d record(s) with status null (no value) and %d nan/inf left out of min, max, first and last; " +
		"first_record and last_record are the raw ends when they differ"
	excludedNullKey   = "null"
	excludedNonFinite = "nan_inf"
)

// statusIsNull says whether a rendered BStatus ("{alarm,null}", "{null} @ facets") has the null flag.
func statusIsNull(status string) bool {
	open, end := strings.IndexByte(status, '{'), strings.IndexByte(status, '}')
	if open < 0 || end < open {
		return false
	}
	for _, flag := range strings.Split(status[open+1:end], ",") {
		if strings.TrimSpace(flag) == statusNullTag {
			return true
		}
	}
	return false
}

// extremes tracks what the summary compares: min, max and the first and last compared record,
// beside the raw first and last record and the excluded counts.
type extremes struct {
	minRec, maxRec, firstRec, lastRec, rawFirst, rawLast *pyjson.Object
	minVal, maxVal                                       float64
	nulls, nonFinite                                     int
}

// add takes one record, newest first; isNum says its value parsed as a number.
func (e *extremes) add(rec *pyjson.Object, status string, value float64, isNum bool) {
	if e.rawLast == nil {
		e.rawLast = rec
	}
	e.rawFirst = rec
	switch {
	case statusIsNull(status):
		e.nulls++
		return
	case isNum && (math.IsNaN(value) || math.IsInf(value, 0)):
		e.nonFinite++
		return
	}
	if e.lastRec == nil {
		e.lastRec = rec
	}
	e.firstRec = rec
	if !isNum {
		return
	}
	// A tie keeps the earliest record: when the value was first reached.
	if e.minRec == nil || value <= e.minVal {
		e.minRec, e.minVal = rec, value
	}
	if e.maxRec == nil || value >= e.maxVal {
		e.maxRec, e.maxVal = rec, value
	}
}

// set writes first and last, and min and max when numeric says they apply.
func (e *extremes) set(out *pyjson.Object, numeric bool) {
	out.Set("first", nilIfNone(e.firstRec))
	out.Set("last", nilIfNone(e.lastRec))
	if numeric && e.minRec != nil {
		minTs, _ := e.minRec.Get("timestamp")
		maxTs, _ := e.maxRec.Get("timestamp")
		out.Set("min", obj("value", e.minVal, "timestamp", minTs))
		out.Set("max", obj("value", e.maxVal, "timestamp", maxTs))
	}
}

// setExcluded adds excluded_from_extremes, the raw ends that differ from first and last, and the
// note, when any record was left out; it goes after status_counts.
func (e *extremes) setExcluded(out *pyjson.Object) {
	if e.nulls+e.nonFinite == 0 {
		return
	}
	out.Set("excluded_from_extremes", obj(excludedNullKey, e.nulls, excludedNonFinite, e.nonFinite))
	if e.rawFirst != e.firstRec {
		out.Set("first_record", e.rawFirst)
	}
	if e.rawLast != e.lastRec {
		out.Set("last_record", e.rawLast)
	}
	out.Set("note", fmt.Sprintf(msgSummaryExcl, e.nulls, e.nonFinite))
}
