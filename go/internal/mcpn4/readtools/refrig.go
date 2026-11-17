package readtools

import (
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Refrigeration fault rules over history data (opportunity O1), report only: every threshold is
// an argument the caller gives (the sources disagree, e.g. a short cycle is a run under ~2 min
// for some and under 10 min for others), every history read is one GET, and a rule whose
// history could not be read is not_checked, never clean.

// Bounds of n4_refrigeration_faults.
const (
	RefrigDefaultRows  = 5000
	RefrigDefaultLimit = 50
)

// refrigRun is one call: its window, bounds, and the histories already read (by id).
type refrigRun struct {
	t                       *Tools
	params                  string
	maxRows, timeout, limit int
	end                     time.Time
	read                    map[string]*refrigSeries
	order                   []string // ids in the order first read: a stable reply
	requests                int
}

// RefrigerationFaults is n4_refrigeration_faults: the rules named in rules, each on its histories.
func (t *Tools) RefrigerationFaults(args *pyjson.Object) (*pyjson.Object, error) {
	params, err := bql.HistoryParams(strArg(args, "period", ""), strArg(args, "start", ""), strArg(args, "end", ""))
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	rules, _ := lookup(args, "rules").(*pyjson.Object)
	if rules == nil || rules.Len() == 0 {
		return nil, toolError("rules must name at least one of %s", refrigRuleNames)
	}
	r := &refrigRun{t: t, params: params, maxRows: intArg(args, "max_rows", RefrigDefaultRows),
		timeout: intArg(args, "timeout_s", 60), limit: intArg(args, "limit", RefrigDefaultLimit),
		end: time.Now(), read: map[string]*refrigSeries{}}
	if e, err := time.Parse(time.RFC3339Nano, strArg(args, "end", "")); err == nil {
		r.end = e // an open interval at the window end is cut there, not at now
	}
	out := obj("range", params)
	summary := pyjson.NewObject()
	for _, name := range rules.Keys() {
		spec, _ := lookup(rules, name).(*pyjson.Object)
		check := refrigChecks[name]
		section, err := check(r, spec)
		if err != nil {
			return nil, err
		}
		out.Set(name, section)
		summary.Set(name, refrigSummaryLine(section))
	}
	out.Set("summary", summary)
	out.Set("histories", r.historiesRead())
	out.Set("requests", r.requests)
	return out, nil
}

// series reads history id once per call; a read failure is the series' failed text.
func (r *refrigRun) series(id string) (*refrigSeries, error) {
	if s, ok := r.read[id]; ok {
		return s, nil
	}
	r.requests++
	s, err := r.t.readSeries(id, r.params, r.maxRows, r.timeout)
	if err != nil {
		return nil, err
	}
	r.read[id] = s
	r.order = append(r.order, id)
	return s, nil
}

func (r *refrigRun) historiesRead() []any {
	out := []any{}
	for _, id := range r.order {
		s := r.read[id]
		o := obj("id", s.id, "records", s.records, "null_records", s.nulls, "truncated", s.truncated)
		if s.failed != "" {
			o.Set("error", s.failed)
		}
		out = append(out, o)
	}
	return out
}

// refrigSummaryLine is "ok", "N finding(s)" or "not checked: why".
func refrigSummaryLine(section *pyjson.Object) string {
	if why, ok := lookup(section, "not_checked").(string); ok {
		return "not checked: " + why
	}
	n, _ := lookup(section, "count").(int)
	if n == 0 {
		return "ok"
	}
	return pyjson.Str(n) + " finding(s)"
}
