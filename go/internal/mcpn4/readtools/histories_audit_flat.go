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
	haFlatDownNote    = "device down, not a placeholder: the point's status has down, stale or fault set"
	haFlatShape       = "the min/max answer has %d row(s) and %d column(s), expected 1 and 2"
)

// flatCheck fills the flat list from the recording histories in exts.
func (r *haRun) flatCheck(exts []histExt) {
	r.readFlatPoints()
	windowStart := r.cfg.now.Add(-r.flatSpan)
	start := windowStart.UTC().Format(windowSent)
	seen := map[string]bool{}
	var todo []histExt
	for _, e := range exts {
		last := strings.TrimSpace(e.last)
		if e.enabled == false || e.id == "" || last == "null" || seen[e.id] {
			continue
		}
		seen[e.id] = true
		if !r.flatInScope(e) {
			continue
		}
		if at, ok := bql.ParseAbsTime(last); ok && at.Before(windowStart) {
			continue // no record in the window: stale says so, there is no value to compare
		}
		todo = append(todo, e)
	}
	// The histories are read haWorkers at a time; the items and failures are taken in the order of
	// exts, so the reply is what one-by-one reads gave.
	results := make([]flatResult, len(todo))
	r.beginPool()
	errs := runPool(len(todo), func(i int) {
		results[i] = r.flatItem(todo[i], start)
		r.finished("flat", len(todo), results[i].reads)
	})
	for i, err := range errs {
		if err != nil {
			results[i] = flatResult{failed: err.Error()}
		}
	}
	for i, e := range todo {
		res := results[i]
		r.requests += res.reads
		r.flatReads += res.reads
		if res.failed != "" {
			r.flatFailed(e.id, res.failed)
		} else if res.item != nil {
			list := "flat"
			if flags, down := r.flatPointDown(e); down {
				list = "flat_down"
				res.item.Set("status", flags)
				res.item.Set("note", haFlatDownNote)
			}
			r.lists[list] = append(r.lists[list], res.item)
		}
	}
}

// flatResult is the outcome of one history's flat check: its item (nil when not flat), the
// station reads it made and, when a read failed, why.
type flatResult struct {
	item   *pyjson.Object
	reads  int
	failed string
}

// flatItem is e's flat item when the window [start, now] holds one value and records exist
// before it; no item otherwise, or a failure reason when a read failed.
func (r *haRun) flatItem(e histExt, start string) flatResult {
	value, reads, err := r.windowValue(e.id, start)
	if err != nil {
		return flatResult{reads: reads, failed: err.Error()}
	}
	if value == "" {
		return flatResult{reads: reads}
	}
	before, err := bql.HistoryParams("", "", start)
	if err != nil {
		return flatResult{reads: reads, failed: err.Error()}
	}
	reads++
	total, note, _ := r.t.countHistory(e.id, before, r.timeout)
	n, ok := total.(int)
	if !ok {
		return flatResult{reads: reads, failed: note}
	}
	if n == 0 {
		return flatResult{reads: reads} // younger than the window: not one value for more than flat_days
	}
	return flatResult{reads: reads, item: obj("history_id", e.id, "ext", e.path, "point", parentPath(e.path),
		"value", value, "window_start", start, "note", fmt.Sprintf(haFlatNote, spanText(r.flatSpan)))}
}

// windowValue is the one value of id's records since start, as the station renders it; "" when
// the window holds no record or more than one value. reads is the station reads it made.
func (r *haRun) windowValue(id, start string) (value string, reads int, err error) {
	params, err := bql.HistoryParams("", start, "")
	if err != nil {
		return "", 0, err
	}
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistoryAggregateQuery)
	if err != nil {
		return "", 0, err
	}
	columns, rows, _, _, err := r.t.fetchHistory(ord, 1, r.timeout)
	if err != nil {
		return "", 1, err
	}
	if len(rows) != 1 || len(columns) < 2 {
		return "", 1, fmt.Errorf(haFlatShape, len(rows), len(columns))
	}
	lo, hi := strings.TrimSpace(cell(rows[0], columns[0])), strings.TrimSpace(cell(rows[0], columns[1]))
	if lo == "" || lo == "null" || lo != hi {
		return "", 1, nil
	}
	return lo, 1, nil
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
	r.flatSensors = strArg(args, "flat_scope", "all") == "sensors"
	r.flatDown = boolArg(args, "flat_down", false)
}

// flatReply adds the flat list's arguments, read count and failure note to out.
func (r *haRun) flatReply(out *pyjson.Object) {
	out.Set("flat_days", r.flatDays)
	out.Set("flat_reads", r.flatReads)
	if r.flatSensors {
		out.Set("flat_scope", "sensors")
	}
	if r.flatFails > 0 {
		r.notes = append(r.notes, fmt.Sprintf(haFlatFailNote, r.flatFails, r.flatNote))
	}
}
