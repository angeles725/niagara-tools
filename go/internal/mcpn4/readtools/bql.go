package readtools

import (
	"errors"
	"math"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/session"
)

// The bulk reads n4_bql_query and n4_inventory (tools_read.py:531-688, retro 2026-10-02 D1, D7,
// D8): one projected BQL GET per query, parsed as ITableToCsv.

// inventoryQueries is INVENTORY_QUERIES: each step and its query, in order.
var inventoryQueries = [][2]string{
	{"networks", "select slotPath, name, type from driver:DeviceNetwork"},
	{"devices", "select slotPath, name, type from driver:Device"},
	{"points", "select slotPath, name, type, out from control:ControlPoint"},
}

// inventoryColumns is INVENTORY_COLUMNS: the keys summarize_inventory reads, in the order every
// inventory query selects them.
var inventoryColumns = []string{"Slot Path", "Name", "Type"}

// recordRead is _record_read: a session observation of a read naming args[key] (no audit log
// runs without --allow-writes, which this version refuses).
func (t *Tools) recordRead(tool, key string, args *pyjson.Object, rows int) {
	ord := ""
	if v, ok := args.Get(key); ok {
		ord = pyjson.Str(v)
	}
	t.Ctx.Observations = append(t.Ctx.Observations, obj("ts", session.Now(), "tool", tool, "rows", rows,
		"ord", t.Ctx.Scrub(ord)))
}

// progress is _progress: one entry in log and, best effort, a line in the operator's file.
func (t *Tools) progress(tool, step string, rows, elapsed int, log []any) []any {
	entry := obj("ts", session.Now(), "tool", tool, "step", step, "rows", rows, "elapsed_ms", elapsed)
	if path := t.Ctx.ProgressPath; path != "" {
		if f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o666); err == nil {
			line, _ := pyjson.Marshal(entry, pyjson.Options{})
			f.Write(append(line, '\n'))
			f.Close()
		}
	}
	return append(log, entry)
}

// query runs one projected BQL GET: the columns, rows, whether they were cut (byte or row
// cap) and the latency in whole milliseconds.
func (t *Tools) query(base, query string, maxRows, timeout int) ([]string, []*pyjson.Object, bool, int, error) {
	ord, err := bql.ComposeOrd(base, query)
	if err != nil {
		return nil, nil, false, 0, toolError("%s", err.Error())
	}
	start := t.Ctx.Clock()
	text, cut, err := t.Ctx.Session.Client.(Client).GetOrd(ord, float64(timeout), bql.MaxResponseBytes)
	if err != nil {
		return nil, nil, false, 0, err
	}
	if cut { // the byte cap may fall inside a quoted cell: keep whole records only
		text = bql.CompleteRecords(text)
	}
	var positions []int // localized headers: the name column is decoded by position (F12)
	if slots := bql.SelectedSlots(query); slots != nil {
		positions = []int{}
		for i, n := range slots {
			if strings.ToLower(n) == "name" {
				positions = append(positions, i)
			}
		}
	}
	columns, rows, capped, err := bql.ParseCSV(text, maxRows, positions)
	var ve *pyjson.ValueError
	if errors.As(err, &ve) { // no header or a ragged record: never a guessed table
		return nil, nil, false, 0, toolError("%s (query: %s)", ve.Msg, bql.Strip(query))
	}
	if err != nil {
		return nil, nil, false, 0, err
	}
	return columns, rows, cut || capped, int(math.RoundToEven((t.Ctx.Clock() - start) * 1000)), nil
}

// BqlQuery is n4_bql_query: one select over base; the rows go into the reply, or into a private
// result file under the state dir when output_file names one.
func (t *Tools) BqlQuery(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	path := ""
	if v, ok := args.Get("output_file"); ok {
		var err error
		if path, err = t.outputPath(v.(string), "bql", bqlOutputFile, "ending in .csv or .json "); err != nil {
			return nil, err
		}
	}
	q := strArg(args, "query", "")
	columns, rows, truncated, elapsed, err := t.query(base, q, intArg(args, "max_rows", bql.DefaultMaxRows),
		intArg(args, "timeout_s", 60))
	if err != nil {
		return nil, err
	}
	t.recordRead("n4_bql_query", "base", args, len(rows))
	out := obj("base", base, "query", bql.Strip(q), "columns", strs(columns), "row_count", len(rows),
		"truncated", truncated, "elapsed_ms", elapsed)
	if path == "" {
		out.Set("rows", list(rows))
		return out, nil
	}
	if err := writeRows(path, base, bql.Strip(q), columns, rows); err != nil {
		var ve *pyjson.ValueError
		if errors.As(err, &ve) {
			return nil, err
		}
		return nil, writeFailed(path, err)
	}
	out.Set("output_file", path)
	return out, nil
}

// Inventory is n4_inventory: networks, devices and points below base through three queries,
// counted per network and device, each step logged as progress.
func (t *Tools) Inventory(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", "station:|slot:/Drivers")
	maxRows, timeout := intArg(args, "max_rows", bql.DefaultMaxRows), intArg(args, "timeout_s", 60)
	results, keyed := map[string][]*pyjson.Object{}, map[string][]*pyjson.Object{}
	var log []any
	truncated, total, rowCount := false, 0, 0
	for _, step := range inventoryQueries {
		columns, rows, cut, elapsed, err := t.query(base, step[1], maxRows, timeout)
		if err != nil {
			return nil, err
		}
		results[step[0]], truncated, total, rowCount = rows, truncated || cut, total+elapsed, rowCount+len(rows)
		log = t.progress("n4_inventory", step[0], len(rows), elapsed, log)
		if keyed[step[0]], err = bql.ByPosition(columns, rows, inventoryColumns); err != nil {
			return nil, toolError("%s query: %s", step[0], err.Error())
		}
	}
	out, err := bql.SummarizeInventory(base, keyed["networks"], keyed["devices"], keyed["points"])
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	t.recordRead("n4_inventory", "base", args, rowCount)
	out.Set("base", base)
	out.Set("truncated", truncated)
	out.Set("progress", log)
	out.Set("elapsed_ms", total)
	if v, _ := args.Get("include_points"); v == true {
		out.Set("point_rows", list(results["points"]))
	}
	return out, nil
}

// list is rows as the []any pyjson encodes.
func list(rows []*pyjson.Object) []any {
	out := make([]any, len(rows))
	for i, r := range rows {
		out[i] = r
	}
	return out
}

// strs is a list of str as pyjson holds it.
func strs(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}

// strArg is args.get(key, def) for a string the schema checked.
func strArg(args *pyjson.Object, key, def string) string {
	if v, ok := args.Get(key); ok {
		return v.(string)
	}
	return def
}
