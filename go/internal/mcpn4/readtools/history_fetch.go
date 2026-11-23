package readtools

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The station side of n4_read_history: one history ORD read as ITableToCsv, the record count,
// and an extension ORD resolved to its history id.

// historyExtQuery reads each history extension's slot path and history id, by position.
const historyExtQuery = "select slotPath, historyConfig.id from history:HistoryExt"

// historyRowBytes bounds the bytes read per requested record (the records come newest first,
// so a cut keeps the newest whole records).
const historyRowBytes = 2048

// Messages of n4_read_history, as data.
const (
	msgOneSource  = "give exactly one of history_id (e.g. /MyStation/MyPoint) or ext_ord (a station:|slot:/... history extension ORD)"
	msgUnreadable = "history %s could not be read: %s. If the id does not exist on the station, list the history ids " +
		"with n4_bql_query query 'select historyConfig.id from history:HistoryExt'"
	msgNotExt      = "ext_ord %s is not a history extension: no history:HistoryExt row answers that slot (query: %s)"
	msgExtRoot     = "ext_ord must name a history extension below the root, got %s"
	msgCountFailed = "the record count (select count(*)) failed: %s"
	msgCountShape  = "the record count (select count(*)) answered no whole number"
	// msgFilterEmpty is the warning of a filtered read of no rows whose count did not answer (retro
	// R8 ΔN2: a where that names no field answers zero rows with no error).
	msgFilterEmpty = "no rows came back and the record count (select count(*)) did not answer: " +
		"the filter may name no field, so this empty list does not show that no record matches"
)

// warnEmptyFilter sets the reply's "warning" when a filtered read (where not empty) returned no
// rows and its count is unknown; any other read has no such warning.
func warnEmptyFilter(out *pyjson.Object, where string, rows int, total any) {
	if where != "" && rows == 0 && total == nil {
		out.Set("warning", msgFilterEmpty)
	}
}

// fetchHistory is one GET of a history ORD: the columns, rows (at most maxRows), whether more
// records exist than were returned, and the latency in whole milliseconds.
func (t *Tools) fetchHistory(ord string, maxRows, timeout int) ([]string, []*pyjson.Object, bool, int, error) {
	start := t.Ctx.Clock()
	maxBytes := min(bql.MaxResponseBytes, 4096+maxRows*historyRowBytes)
	text, cut, err := t.Ctx.Session.Client.(Client).GetOrd(ord, float64(timeout), maxBytes)
	if err != nil {
		return nil, nil, false, 0, err
	}
	if cut { // the byte cap may fall inside a quoted cell: keep whole records only
		text = bql.CompleteRecords(text)
	}
	columns, rows, capped, err := bql.ParseCSV(text, maxRows, []int{})
	if err != nil {
		return nil, nil, false, 0, err
	}
	return columns, rows, cut || capped, int(math.RoundToEven((t.Ctx.Clock() - start) * 1000)), nil
}

// unreadable is a failed records read as the reply names it: an AuthError passes unchanged
// (the session latches on it); any other station answer names the id and how to list ids.
func unreadable(id string, err error) error {
	var be *box.Error
	var ve *pyjson.ValueError
	if errors.As(err, &be) && be.IsAuth() || !errors.As(err, &be) && !errors.As(err, &ve) {
		return err
	}
	return toolError(msgUnreadable, id, err.Error())
}

// countHistory is the number of records the range holds, or nil and a note saying why not.
func (t *Tools) countHistory(id, params string, timeout int) (any, string, int) {
	return t.countHistoryWhere(id, params, "", timeout)
}

// countHistoryWhere is the number of records of the range that match where ("" for all).
func (t *Tools) countHistoryWhere(id, params, where string, timeout int) (any, string, int) {
	ord, err := bql.ComposeFilteredHistoryOrd(id, params, bql.HistoryCountQuery, where, "")
	if err != nil {
		return nil, err.Error(), 0
	}
	_, rows, _, elapsed, err := t.fetchHistory(ord, 1, timeout)
	if err != nil {
		return nil, fmt.Sprintf(msgCountFailed, err.Error()), elapsed
	}
	if len(rows) == 1 && len(rows[0].Keys()) == 1 {
		v, _ := rows[0].Get(rows[0].Keys()[0])
		if n, err := strconv.Atoi(strings.TrimSpace(pyjson.Str(v))); err == nil && n >= 0 {
			return n, "", elapsed
		}
	}
	return nil, msgCountShape, elapsed
}

// resolveHistoryExt is the history id of the extension at extOrd: one BQL select at the
// component holding it, matched by the extension's own slot path.
func (t *Tools) resolveHistoryExt(extOrd string, timeout int) (string, int, error) {
	slot, err := bql.BaseSlot(extOrd)
	if err != nil {
		return "", 0, toolError("%s", err.Error())
	}
	cut := strings.LastIndexByte(slot, '/')
	if slot == "/" || cut < 0 {
		return "", 0, toolError(msgExtRoot, pyjson.StrRepr(extOrd))
	}
	parent := slot[:cut]
	if parent == "" {
		parent = "/"
	}
	columns, rows, _, elapsed, err := t.query(bql.StationSlot+parent, historyExtQuery, bql.DefaultMaxRows, timeout)
	if err != nil {
		return "", elapsed, err
	}
	keyed, err := bql.ByPosition(columns, rows, []string{"slot", "id"})
	if err != nil {
		return "", elapsed, toolError("%s", err.Error())
	}
	for _, row := range keyed {
		if s, _ := row.Get("slot"); s == "slot:"+slot {
			id, _ := row.Get("id")
			return pyjson.Str(id), elapsed, nil
		}
	}
	return "", elapsed, toolError(msgNotExt, pyjson.StrRepr(extOrd), historyExtQuery)
}
