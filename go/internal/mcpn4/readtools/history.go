package readtools

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_history: the last records of one history, named by its id or by its extension's ORD,
// so a rollout proves records arrive and not only that the extension is configured. Read-only:
// fixed queries over a history: ORD, never a caller's query, never a write.

// historyFields name the record columns by position (a localized station translates headers).
var historyFields = []string{"timestamp", "value", "status"}

// bareHistoryNameRe is a history_id given as a bare history name (no '/'), resolved against
// the session's station; ValidateHistoryId still checks the composed id.
var bareHistoryNameRe = regexp.MustCompile(`^[^/]+$`)

// Defaults of n4_read_history (the registry schema caps limit at 1000).
const (
	historyDefaultLimit   = 20
	historyDefaultTimeout = 60
)

// ReadHistory is n4_read_history.
func (t *Tools) ReadHistory(args *pyjson.Object) (*pyjson.Object, error) {
	_, hasID := args.Get("history_id")
	extOrd, hasExt := args.Get("ext_ord")
	if hasID == hasExt {
		return nil, toolError("%s", msgOneSource)
	}
	period := strArg(args, "period", "")
	params, err := bql.HistoryParams(period, strArg(args, "start", ""), strArg(args, "end", ""))
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	limit, timeout := intArg(args, "limit", historyDefaultLimit), intArg(args, "timeout_s", historyDefaultTimeout)
	id, key, elapsed := strArg(args, "history_id", ""), "history_id", 0
	if hasID && bareHistoryNameRe.MatchString(id) { // a bare name is a history of the session's station
		id = "/" + t.Ctx.Session.StationName + "/" + id
	}
	if hasExt {
		key = "ext_ord"
		if id, elapsed, err = t.resolveHistoryExt(extOrd.(string), timeout); err != nil {
			return nil, err
		}
	}
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistoryRecordsQuery)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	columns, rows, truncated, ms, err := t.fetchHistory(ord, limit, timeout)
	if err != nil {
		return nil, unreadable(id, err)
	}
	keyed, err := bql.ByPosition(columns, rows, historyFields)
	if err != nil {
		return nil, unreadable(id, err)
	}
	total, note, countMs := t.countHistory(id, params, timeout)
	var summary any
	summaryNote, summaryMs := "", 0
	if v, ok := args.Get("summary"); ok && pyjson.Truthy(v) {
		maxRows := intArg(args, "summary_max_rows", summaryDefaultRows)
		summary, summaryNote, summaryMs = t.historySummary(id, params, maxRows, timeout, total)
	}
	t.recordRead("n4_read_history", key, args, len(rows))
	out := obj("history_id", id)
	if hasExt {
		out.Set("ext_ord", extOrd)
	}
	for _, k := range []string{"period", "start", "end"} {
		if v, ok := args.Get(k); ok {
			out.Set(k, v)
		}
	}
	if params != "" {
		out.Set("range", params)
	}
	out.Set("columns", strs(columns))
	out.Set("rows", list(keyed))
	out.Set("row_count", len(rows))
	out.Set("order", "newest_first")
	out.Set("truncated", truncated)
	out.Set("total_records", total)
	if note != "" {
		out.Set("total_records_note", note)
	}
	if summary != nil || summaryNote != "" {
		out.Set("summary", summary)
	}
	if summaryNote != "" {
		out.Set("summary_note", summaryNote)
	}
	out.Set("elapsed_ms", elapsed+ms+countMs+summaryMs)
	return out, nil
}
