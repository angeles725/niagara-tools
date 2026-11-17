package readtools

import (
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_audit (station-audits S7, "who changed overAmpsDelay from 5 s to 3 s"): the audit
// trail of the connected station, newest first, narrowed by component, slot, user, operation
// and time range. Read-only: one fixed query and its count over the history: ORD of
// /<station>/AuditHistory, every filter value checked before any request.

// auditFields name the audit record columns by position (a localized station translates headers).
var auditFields = []string{"timestamp", "operation", "target", "slot", "old_value", "value", "user"}

// Defaults of n4_read_audit (the registry schema caps limit at 1000).
const (
	auditDefaultLimit   = 100
	auditDefaultTimeout = 60
)

// msgAuditUnreadable names the audit history and where its service lives.
const msgAuditUnreadable = "the audit history %s could not be read: %s. The station writes it only while " +
	"Config/Services/AuditHistoryService is enabled"

// ReadAudit is n4_read_audit.
func (t *Tools) ReadAudit(args *pyjson.Object) (*pyjson.Object, error) {
	where, err := bql.AuditWhere(bql.AuditFilter{Target: strArg(args, "target", ""),
		TargetGlob: strArg(args, "target_glob", ""), Slot: strArg(args, "slot", ""),
		User: strArg(args, "user", ""), Operation: strArg(args, "operation", "")})
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	period := strArg(args, "period", "")
	params, err := bql.HistoryParams(period, strArg(args, "start", ""), strArg(args, "end", ""))
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	id, err := bql.ResolveHistoryId(bql.AuditHistoryName, t.Ctx.Session.StationName)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	ord, err := bql.ComposeFilteredHistoryOrd(id, params, bql.AuditRecordsQuery, where, bql.AuditOrder)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	limit, timeout := intArg(args, "limit", auditDefaultLimit), intArg(args, "timeout_s", auditDefaultTimeout)
	columns, rows, truncated, ms, err := t.fetchHistory(ord, limit, timeout)
	if err != nil {
		return nil, auditUnreadable(id, err)
	}
	keyed, err := bql.ByPosition(columns, rows, auditFields)
	if err != nil {
		return nil, auditUnreadable(id, err)
	}
	for _, row := range keyed {
		row.Set("timestamp", bql.IsoOrNull(cell(row, "timestamp")))
	}
	total, note, countMs := t.countHistoryWhere(id, params, where, timeout)
	t.recordRead("n4_read_audit", "target", args, len(rows))
	out := obj("history_id", id, "where", where)
	for _, k := range []string{"period", "start", "end"} {
		if v, ok := args.Get(k); ok {
			out.Set(k, v)
		}
	}
	if params != "" && period == "" { // the resolved start/end; a period is already echoed
		out.Set("range", params)
	}
	out.Set("rows", list(keyed))
	out.Set("row_count", len(rows))
	out.Set("order", "newest_first")
	out.Set("truncated", truncated)
	out.Set("total_records", total)
	if note != "" {
		out.Set("total_records_note", note)
	}
	out.Set("elapsed_ms", ms+countMs)
	return out, nil
}

// auditUnreadable is unreadable with the audit history's own hint: an AuthError (or any error
// unreadable passes unchanged) stays as it is.
func auditUnreadable(id string, err error) error {
	if e := unreadable(id, err); e == err {
		return err
	}
	return toolError(msgAuditUnreadable, id, err.Error())
}
