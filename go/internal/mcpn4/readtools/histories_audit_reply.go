package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The findings of n4_histories_audit from its reads, and the reply that carries them.

// classify fills the finding lists: disabled and stale from each row, colliding ids across them,
// near capacity from one record count per distinct id, and no_history when the points were read.
func (r *haRun) classify(points []string, exts []histExt) {
	add := func(list string, item *pyjson.Object) { r.lists[list] = append(r.lists[list], item) }
	if r.pointsRead {
		for _, p := range pointsWithoutHistory(points, exts) {
			add("no_history", obj("point", p))
		}
	}
	counted := map[string]bool{}
	var toCount []histExt // the record counts to read, in extension order
	for _, e := range exts {
		switch e.enabled {
		case false:
			add("disabled", obj("ext", e.path, "history_id", nilIfEmpty(e.id)))
			continue
		case nil:
			r.notAnswered["enabled"]++
		}
		if e.id == "" {
			continue // an enabled extension with no id has never registered a history
		}
		if item, why := r.cfg.stale(e); item != nil {
			add("stale", item)
		} else if why != "" {
			r.notAnswered[why]++
		}
		if !needsCount(e) {
			if _, ok := bql.ParseCapacity(e.capacity); !ok {
				r.notAnswered["capacity"]++
			}
			continue
		}
		if counted[e.id] {
			continue
		}
		counted[e.id] = true
		toCount = append(toCount, e)
	}
	r.nearCapacity(toCount, add)
	r.lists["colliding_ids"] = collidingIds(exts)
}

// countResult is one record count read: the count, or why there is none.
type countResult struct {
	records int
	note    string
	ok      bool
}

// nearCapacity reads the record count of each of exts, haWorkers at a time, and adds the
// near-capacity items in the order of exts; the first failed read (in that order) is the one noted.
func (r *haRun) nearCapacity(exts []histExt, add func(string, *pyjson.Object)) {
	results := make([]countResult, len(exts))
	runPool(len(exts), func(i int) {
		total, note, _ := r.t.countHistory(exts[i].id, "", r.timeout)
		records, ok := total.(int)
		results[i] = countResult{records, note, ok}
	})
	for i, e := range exts {
		r.requests++
		r.countReads++
		if !results[i].ok {
			if r.countFails++; r.countNote == "" {
				r.countNote = e.id + ": " + results[i].note
			}
			continue
		}
		if item := r.cfg.nearCapacity(e, results[i].records); item != nil {
			add("near_capacity", item)
		}
	}
}

// reply is the audit's reply: its arguments, every list capped at haListCap, the totals, notes.
func (r *haRun) reply(base, span, minSpan string, extRows int) *pyjson.Object {
	out := obj("base", base, "stale_factor", r.cfg.staleFactor, "stale_min", minSpan, "cov_max_age", span,
		"capacity_warn", r.cfg.warn, "exclude_services", r.excludeServices, "services_excluded", r.servicesExcluded)
	counts, cut := obj("points", nil, "history_exts", extRows), []string{}
	names := haLists
	if r.flat {
		names = append(names[:len(names):len(names)], "flat")
		if r.flatDown {
			names = append(names, "flat_down")
		}
	}
	for _, name := range names {
		items := r.lists[name]
		if items == nil {
			items = []*pyjson.Object{}
		}
		if len(items) > haListCap {
			cut = append(cut, name)
			items = items[:haListCap]
		}
		out.Set(name, list(items))
		counts.Set(name, len(r.lists[name]))
	}
	if r.pointCount >= 0 {
		counts.Set("points", r.pointCount-r.servicesExcluded)
	}
	if !r.pointsRead {
		counts.Set("no_history", nil)
		out.Set("no_history_note", r.noHistoryNote)
	}
	out.Set("counts", counts)
	if r.flat {
		r.flatReply(out)
	}
	if len(cut) > 0 {
		r.notes = append(r.notes, fmt.Sprintf(haListsCut, strings.Join(cut, ", ")))
	}
	if r.countFails > 0 {
		r.notes = append(r.notes, fmt.Sprintf(haCountNote, r.countFails, r.countNote))
	}
	if len(r.notAnswered) > 0 {
		na := pyjson.NewObject()
		for _, k := range []string{"enabled", "interval", "last_record", "capacity"} {
			if n := r.notAnswered[k]; n > 0 {
				na.Set(k, n)
			}
		}
		out.Set("not_answered", na)
		r.notes = append(r.notes, haNoAnswer)
	}
	if len(r.notes) > 0 {
		out.Set("notes", r.notes)
	}
	out.Set("truncated", r.extCut || r.pointsCut)
	out.Set("count_reads", r.countReads)
	out.Set("requests", r.requests)
	return out
}
