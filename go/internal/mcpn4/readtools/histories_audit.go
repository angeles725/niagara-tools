package readtools

import (
	"fmt"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_histories_audit (station-audits S1): what is wrong with the histories below a base, in one
// read-only call: two fixed BQL reads (the history extensions with their settings and last
// record, and the control points), then one select count(*) per distinct record-capped history
// id that has recorded, at most max_histories of them.

// haRun is one audit: its arguments, the station reads and the findings so far.
type haRun struct {
	t                  *Tools
	cfg                histAuditCfg
	scope              string
	maxHistories       int
	timeout, requests  int
	countReads         int
	countFails         int
	countNote          string
	notes              []any
	notAnswered        map[string]int
	lists              map[string][]*pyjson.Object
	totals             map[string]any
	extCut, pointsRead bool
	pointsCut          bool
	pointCount         int // -1 when the points read failed
	noHistoryNote      string
	excludeServices    bool
	servicesExcluded   int
}

// HistoriesAudit is n4_histories_audit.
func (t *Tools) HistoriesAudit(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	slot, err := bql.BaseSlot(base)
	if err != nil {
		return nil, toolError(haBadBase, pyjson.StrRepr(base))
	}
	span, minSpan := strArg(args, "cov_max_age", haDefaultCovMaxAge), strArg(args, "stale_min", haDefaultStaleMin)
	covMaxAge, ok := bql.ParseSpan(span)
	if !ok {
		return nil, toolError(haBadSpan, "cov_max_age", pyjson.StrRepr(span))
	}
	staleMin, ok := bql.ParseSpan(minSpan)
	if !ok {
		return nil, toolError(haBadSpan, "stale_min", pyjson.StrRepr(minSpan))
	}
	r := &haRun{t: t, scope: bql.StationSlot + slot, notAnswered: map[string]int{},
		maxHistories: intArg(args, "max_histories", haDefaultMaxHistories),
		timeout:      intArg(args, "timeout_s", haDefaultTimeout),
		lists:        map[string][]*pyjson.Object{}, totals: map[string]any{}, notes: []any{},
		excludeServices: boolArg(args, "exclude_services", true),
		cfg: histAuditCfg{staleFactor: floatArg(args, "stale_factor", haDefaultStaleFactor), covMaxAge: covMaxAge,
			staleMin: staleMin,
			warn:     floatArg(args, "capacity_warn", haDefaultCapacityWarn), now: bql.Now()}}
	start := t.Ctx.Clock()
	exts, err := r.readExts()
	if err != nil {
		return nil, err
	}
	points, err := r.readPoints()
	if err != nil {
		return nil, err
	}
	if r.excludeServices {
		points, r.servicesExcluded = withoutServices(points)
	}
	r.classify(points, exts)
	out := r.reply(base, span, minSpan, len(exts))
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	t.recordRead("n4_histories_audit", "base", args, len(exts)+len(points))
	return out, nil
}

// readExts reads the history extensions below the scope; a failed read fails the call.
func (r *haRun) readExts() ([]histExt, error) {
	r.requests++
	columns, raw, cut, _, err := r.t.query(r.scope, bql.HistoriesAuditExtQuery, r.maxHistories, r.timeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, haExtFields)
	}
	if err != nil {
		section, err := r.t.sectionError(err)
		if err != nil {
			return nil, err
		}
		return nil, toolError(haExtFailed, bql.HistoriesAuditExtQuery, lookup(section, "error"))
	}
	if r.extCut = cut; cut {
		r.notes = append(r.notes, fmt.Sprintf(haCutRows, len(rows)))
	}
	exts := make([]histExt, 0, len(rows))
	for _, row := range rows {
		exts = append(exts, histExt{path: cellText(row, "path"), typ: cellText(row, "type"),
			enabled: enabledCell(cellText(row, "enabled")), id: strings.TrimSpace(cellText(row, "id")),
			capacity: cellText(row, "capacity"), policy: cellText(row, "policy"),
			interval: cellText(row, "interval"), last: cellText(row, "last")})
	}
	return exts, nil
}

// readPoints reads the control points below the scope; a failed or cut read leaves no_history
// unchecked (nil) with a note, never an asserted absence.
func (r *haRun) readPoints() ([]string, error) {
	r.requests++
	columns, raw, cut, _, err := r.t.query(r.scope, bql.HistoriesAuditPointQuery, haMaxPoints, r.timeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, haPointFields)
	}
	if err != nil {
		section, err := r.t.sectionError(err)
		if err != nil {
			return nil, err
		}
		r.pointCount = -1
		r.noHistoryNote = fmt.Sprintf(haPointsFailed, bql.HistoriesAuditPointQuery, lookup(section, "error"))
		return nil, nil
	}
	points := make([]string, 0, len(rows))
	for _, row := range rows {
		points = append(points, cellText(row, "path"))
	}
	r.pointCount, r.pointsCut, r.pointsRead = len(points), cut, !cut && !r.extCut
	switch {
	case cut:
		r.noHistoryNote = fmt.Sprintf(haCutAbsence, "control points", len(rows))
	case r.extCut:
		r.noHistoryNote = fmt.Sprintf(haCutAbsence, "history extensions", r.maxHistories)
	}
	return points, nil
}

// floatArg is args.get(key, def) for a number the schema bounds.
func floatArg(args *pyjson.Object, key string, def float64) float64 {
	if v, ok := args.Get(key); ok {
		if f, err := v.(pyjson.Number).Float64(); err == nil {
			return f
		}
	}
	return def
}

// boolArg is args.get(key, def) for a boolean the schema checked.
func boolArg(args *pyjson.Object, key string, def bool) bool {
	if v, ok := args.Get(key); ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}
