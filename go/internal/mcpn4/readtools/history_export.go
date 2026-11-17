package readtools

import (
	"bufio"
	"math"
	"os"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// n4_export_history (station-audits S2): a range of one or more histories written to ONE private
// CSV file under <state-dir>/exports/, for client reports. The station is only read (the fixed
// machine-encoded summary query of n4_read_history, one GET per history); the file name is the
// server's, never the caller's.

// Limits and messages of n4_export_history, as data.
const (
	exportMaxIds      = 20
	exportDefaultRows = 100000
	msgExportIds      = "history_ids must list 1 to %d history ids (/<station>/<history name>, or a bare history name of the connected station), got %s"
	msgExportShape    = "history %s: the summary read answered %d column(s), expected 3 (timestamp, value, status)"
)

// exportHeader is the first line of an export file.
var exportHeader = []string{"history_id", "timestamp", "value", "status"}

// ExportHistory is n4_export_history.
func (t *Tools) ExportHistory(args *pyjson.Object) (*pyjson.Object, error) {
	ids, err := exportIds(args, t.Ctx.Session.StationName)
	if err != nil {
		return nil, err
	}
	params, err := bql.HistoryParams(strArg(args, "period", ""), strArg(args, "start", ""), strArg(args, "end", ""))
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	dir := t.stateDir()
	if err := safety.CheckStateDir(dir); err != nil {
		return nil, toolError("%s", err.Error())
	}
	maxRows, timeout := intArg(args, "max_rows", exportDefaultRows), intArg(args, "timeout_s", historyDefaultTimeout)
	path := safety.Join(dir, exportSubdir, exportFileName(t.Ctx.Session.StationName, time.Now()))
	if err := exportDir(safety.Dirname(path)); err != nil {
		return nil, exportWriteFailed(path, err)
	}
	start, per, total := t.Ctx.Clock(), []any{}, 0
	var readErr error // a station answer that stopped the export (the file is removed)
	err = writePrivate(path, func(f *os.File) error {
		w := bufio.NewWriter(f)
		w.WriteString(csvLine(exportHeader))
		for _, id := range ids {
			n, cut, err := t.exportOne(w, id, params, maxRows, timeout)
			if err != nil {
				readErr = err
				return err
			}
			per, total = append(per, obj("history_id", id, "rows", n, "truncated", cut)), total+n
		}
		return w.Flush()
	})
	if readErr != nil {
		return nil, readErr
	}
	if err != nil {
		return nil, exportWriteFailed(path, err)
	}
	size := int64(0)
	if info, err := os.Stat(path); err == nil {
		size = info.Size()
	}
	t.recordRead("n4_export_history", "history_ids", args, total)
	out := obj("path", path, "rows_total", total, "per_history", per, "bytes", size)
	for _, k := range []string{"period", "start", "end"} {
		if v, ok := args.Get(k); ok {
			out.Set(k, v)
		}
	}
	if params != "" && strArg(args, "period", "") == "" {
		out.Set("range", params)
	}
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	return out, nil
}

// exportOne reads the newest maxRows records of id's range with machine-encoded time and value
// and writes them oldest first; it returns how many it wrote and whether the range held more.
func (t *Tools) exportOne(w *bufio.Writer, id, params string, maxRows, timeout int) (int, bool, error) {
	ord, err := bql.ComposeHistoryOrd(id, params, bql.HistorySummaryQuery)
	if err != nil {
		return 0, false, toolError("%s", err.Error())
	}
	columns, rows, cut, _, err := t.fetchHistory(ord, maxRows, timeout)
	if err != nil {
		return 0, false, unreadable(id, err)
	}
	if len(rows) > 0 && len(columns) < 3 {
		return 0, false, toolError(msgExportShape, id, len(columns))
	}
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		if _, err := w.WriteString(exportLine(id, cell(row, columns[0]), cell(row, columns[1]), cell(row, columns[2]))); err != nil {
			return 0, false, err
		}
	}
	return len(rows), cut, nil
}

// exportIds is the history_ids argument resolved like n4_read_history's history_id (a bare name
// is a history of station), 1 to exportMaxIds of them, each checked before any request.
func exportIds(args *pyjson.Object, station string) ([]string, error) {
	var raw []any
	if v, ok := args.Get("history_ids"); ok {
		raw, _ = v.([]any)
	}
	if len(raw) == 0 || len(raw) > exportMaxIds {
		return nil, toolError(msgExportIds, exportMaxIds, pyjson.Repr(len(raw))+" ids")
	}
	ids := make([]string, len(raw))
	for i, v := range raw {
		s, ok := v.(string)
		if !ok {
			return nil, toolError(msgExportIds, exportMaxIds, pyjson.Repr(v))
		}
		id, err := bql.ResolveHistoryId(s, station)
		if err != nil {
			return nil, toolError("%s", err.Error())
		}
		ids[i] = id
	}
	return ids, nil
}
