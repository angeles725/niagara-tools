package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_history: the last records of one history, named by its id or by its extension's ORD,
// so a rollout proves records arrive and not only that the extension is configured. Read-only:
// fixed queries over a history: ORD, never a caller's query, never a write.

// historyFields name the record columns by position (a localized station translates headers).
var historyFields = []string{"timestamp", "value", "status"}

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
	params, echo, err := historyWindow(args)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	status, where, err := historyStatus(args)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	limit, timeout := intArg(args, "limit", historyDefaultLimit), intArg(args, "timeout_s", historyDefaultTimeout)
	id, key, elapsed := strArg(args, "history_id", ""), "history_id", 0
	if hasID {
		if id, err = bql.ResolveHistoryId(id, t.Ctx.Session.StationName); err != nil {
			return nil, toolError("%s", err.Error())
		}
	}
	if hasExt {
		key = "ext_ord"
		if id, elapsed, err = t.resolveHistoryExt(extOrd.(string), timeout); err != nil {
			return nil, err
		}
	}
	ord, err := historyRecordsOrd(id, params, where)
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
	total, note, countMs := t.countHistoryWhere(id, params, where, timeout)
	var kind any
	kindNote, kindMs, wantKind := "", 0, truthyArg(args, "history_type")
	if wantKind {
		if kind, kindNote, kindMs, err = t.historyType(id, timeout); err != nil {
			return nil, err
		}
	}
	ex := t.historyExtras(args, id, params, timeout, total)
	t.recordRead("n4_read_history", key, args, len(rows))
	out := obj("history_id", id)
	if hasExt {
		out.Set("ext_ord", extOrd)
	}
	for _, k := range echo {
		if v, ok := args.Get(k); ok {
			out.Set(k, v)
		}
	}
	if params != "" && period == "" { // the resolved start/end; a period is already echoed
		out.Set("range", params)
	}
	if _, ok := args.Get("status"); ok {
		out.Set("status", status)
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
	if wantKind {
		out.Set("history_type", kind)
		if kindNote != "" {
			out.Set("history_type_note", kindNote)
		}
	}
	ex.set(out)
	out.Set("elapsed_ms", elapsed+ms+countMs+kindMs+ex.ms)
	return out, nil
}
