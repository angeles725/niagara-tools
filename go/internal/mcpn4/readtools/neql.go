package readtools

import (
	"errors"
	"math"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/tags"
)

// NEQL bounds: rows by default and at most (the station sets no limit of its own).
const (
	NeqlDefaultRows = 200
	NeqlMaxRows     = 2000
)

// NeqlQuery is n4_neql_query: one GET of base|neql:<query>[|bql:<select>]|view:file:ITableToCsv.
// Without bql the matches come back as station ORDs; with bql as that select's columns and rows.
func (t *Tools) NeqlQuery(args *pyjson.Object) (*pyjson.Object, error) {
	ord, q, sel, err := neqlOrd(args)
	if err != nil {
		return nil, err
	}
	maxRows := intArg(args, "max_rows", NeqlDefaultRows) // 1..NeqlMaxRows: the registry schema refuses others
	start := t.Ctx.Clock()
	text, cut, err := t.Ctx.Session.Client.(Client).GetOrd(ord, float64(intArg(args, "timeout_s", 60)), bql.MaxResponseBytes)
	var be *box.Error
	if errors.As(err, &be) && strings.HasPrefix(be.Msg, "HTTP 400") {
		return nil, toolError("the station rejected the NEQL query %s (HTTP 400): check its syntax "+
			"(namespace:tag, = != < <= > >= like, and/or/not, \"strings\")", pyjson.StrRepr(q))
	}
	if err != nil {
		return nil, err
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, rows, capped, err := bql.ParseCSV(text, maxRows, []int{})
	if err != nil {
		return nil, err
	}
	out := obj("base_ord", strArg(args, "base_ord", RootOrd), "neql", q)
	if sel != "" {
		out.Set("bql", sel)
		out.Set("columns", strs(columns))
		out.Set("rows", list(rows))
	} else {
		ords := []any{}
		for _, r := range rows {
			cell, _ := lookup(r, columns[0]).(string) // one column, its header localized
			ords = append(ords, tags.MatchOrd(cell))
		}
		out.Set("ords", ords)
	}
	out.Set("count", len(rows))
	out.Set("truncated", cut || capped)
	out.Set("elapsed_ms", int(math.RoundToEven((t.Ctx.Clock()-start)*1000)))
	return out, nil
}

// neqlOrd is the ORD to read, the stripped query and select ("" without bql), or the refusal.
func neqlOrd(args *pyjson.Object) (string, string, string, error) {
	slot, err := bql.BaseSlot(strArg(args, "base_ord", RootOrd))
	if err != nil {
		return "", "", "", err
	}
	q, err := tags.ValidateNeql(lookup(args, "neql"))
	if err != nil {
		return "", "", "", err
	}
	ord := bql.StationSlot + slot + "|" + tags.NeqlScheme + q
	sel := ""
	if v, ok := args.Get("bql"); ok && v != nil {
		if sel, err = tags.ValidateSelect(v); err != nil {
			return "", "", "", err
		}
		ord += "|bql:" + sel
	}
	return ord + "|" + bql.View, q, sel, nil
}
