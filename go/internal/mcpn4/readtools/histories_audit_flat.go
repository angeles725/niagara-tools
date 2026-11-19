package readtools

import (
	"fmt"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_histories_audit flat=true (mcp-n4 delta M3): the histories that held one value for more than
// flat_days days, each a possible device placeholder (a device that sends 0.0 for "no sensor").
// Per distinct recording history: one read of the window's min and max (records with a null
// status left out, as n4_read_history's range_min/range_max) and, when they are one value, one
// record count before the window, so a history younger than the window is never flagged.

// Defaults and messages of the flat check, as data.
const (
	haDefaultFlatDays = 2.0
	haFlatNote        = "possible device placeholder: one value for more than %s"
	haFlatFailNote    = "%d flat read(s) failed, those histories are not in flat; first: %s"
	haFlatShape       = "the min/max answer has %d row(s) and %d column(s), expected 1 and 2"
)

// flatCheck fills the flat list from the recording histories in exts.
func (r *haRun) flatCheck(exts []histExt) {
	windowStart := r.cfg.now.Add(-r.flatSpan)
	start := windowStart.UTC().Format(windowSent)
	seen := map[string]bool{}
	for _, e := range exts {
		last := strings.TrimSpace(e.last)
		if e.enabled == false || e.id == "" || last == "null" || seen[e.id] {
			continue
		}
		seen[e.id] = true
		if at, ok := bql.ParseAbsTime(last); ok && at.Before(windowStart) {
			continue // no record in the window: stale says so, there is no value to compare
		}
		if item := r.flatItem(e, start); item != nil {
			r.lists["flat"] = append(r.lists["flat"], item)
		}
	}
}

// flatItem is e's flat item when the window [start, now] holds one value and records exist
// before it; nil otherwise, or when a read failed (counted in the notes).
func (r *haRun) flatItem(e histExt, start string) *pyjson.Object {
	value, err := r.windowValue(e.id, start)
	if err != nil {
		r.flatFailed(e.id, err.Error())
		return nil
	}
	if value == "" {
		return nil
	}
	before, err := bql.HistoryParams("", "", start)
	if err != nil {
		r.flatFailed(e.id, err.Error())
		return nil
	}
	r.requests++
	r.flatReads++
	total, note, _ := r.t.countHistory(e.id, before, r.timeout)
	n, ok := total.(int)
	if !ok {
		r.flatFailed(e.id, note)
		return nil
	}
	if n == 0 {
		return nil // younger than the window: not one value for more than flat_days
	}
	return obj("history_id", e.id, "ext", e.path, "point", parentPath(e.path), "value", value,
		"window_start", start, "note", fmt.Sprintf(haFlatNote, spanText(r.flatSpan)))
}

// windowValue is the one value of id's records since start, as the station renders it; "" when
// the window holds no record or more than one value.
func (r *haRun) windowValue(id, start string) (string, error) {
	params, err := bql.HistoryParams("", start, "")
	if err != nil {
		return "", err
	}
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistoryAggregateQuery)
	if err != nil {
		return "", err
	}
	r.requests++
	r.flatReads++
	columns, rows, _, _, err := r.t.fetchHistory(ord, 1, r.timeout)
	if err != nil {
		return "", err
	}
	if len(rows) != 1 || len(columns) < 2 {
		return "", fmt.Errorf(haFlatShape, len(rows), len(columns))
	}
	lo, hi := strings.TrimSpace(cell(rows[0], columns[0])), strings.TrimSpace(cell(rows[0], columns[1]))
	if lo == "" || lo == "null" || lo != hi {
		return "", nil
	}
	return lo, nil
}

// flatFailed counts a failed flat read, keeping the first reason.
func (r *haRun) flatFailed(id, why string) {
	if r.flatFails++; r.flatNote == "" {
		r.flatNote = id + ": " + why
	}
}

// flatArgs reads flat and flat_days into r.
func (r *haRun) flatArgs(args *pyjson.Object) {
	r.flat = boolArg(args, "flat", false)
	r.flatDays = floatArg(args, "flat_days", haDefaultFlatDays)
	r.flatSpan = time.Duration(r.flatDays * float64(24*time.Hour))
}

// flatReply adds the flat list's arguments, read count and failure note to out.
func (r *haRun) flatReply(out *pyjson.Object) {
	out.Set("flat_days", r.flatDays)
	out.Set("flat_reads", r.flatReads)
	if r.flatFails > 0 {
		r.notes = append(r.notes, fmt.Sprintf(haFlatFailNote, r.flatFails, r.flatNote))
	}
}
