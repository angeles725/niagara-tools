package writeops

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The history reads of n4_remove_history: the record count (n4_read_history's select count(*)
// over the history: ORD) before, and the same read after, where an answer "cannot be read"
// (the station's own refusal, an HTTP status, a reply with no table) means the history is gone.

// readHistoryCount is whether the history can be read, its record count (nil when the read
// answered none) and a note saying why not; an auth failure or a transport failure is an error.
// Only the station's HTTP 404 proves the history is absent (measured on 4.14 for an unknown id and
// for a deleted one); another HTTP status leaves it possibly there.
func readHistoryCount(r ordReader, id string) (bool, any, string, error) {
	exists, count, note, _, err := readHistoryState(r, id)
	return exists, count, note, err
}

// readHistoryState is readHistoryCount plus known: false when the read neither found the history
// nor got the station's 404 for it.
func readHistoryState(r ordReader, id string) (bool, any, string, bool, error) {
	ord, err := bql.ComposeFilteredHistoryOrd(id, "", bql.HistoryCountQuery, "", "")
	if err != nil {
		return false, nil, "", false, plan.ToolError(err.Error())
	}
	text, cut, err := r.GetOrd(ord, ackTimeout, bql.MaxResponseBytes)
	var be *box.Error
	var ve *pyjson.ValueError
	if errors.As(err, &be) && historyNotFound(be) {
		return false, nil, fmt.Sprintf(msgHRCount, err), true, nil
	} else if errors.As(err, &be) && !be.IsAuth() {
		return true, nil, fmt.Sprintf(msgHRCount, err), false, nil
	} else if err != nil {
		return false, nil, "", false, err
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	_, rows, _, err := bql.ParseCSV(text, 1, []int{})
	if errors.As(err, &ve) {
		return true, nil, fmt.Sprintf(msgHRCount, err), false, nil
	} else if err != nil {
		return false, nil, "", false, err
	}
	if len(rows) == 1 && len(rows[0].Keys()) == 1 {
		v, _ := rows[0].Get(rows[0].Keys()[0])
		if n, err := strconv.Atoi(strings.TrimSpace(pyjson.Str(v))); err == nil && n >= 0 {
			return true, n, "", true, nil
		}
	}
	return true, nil, msgHRCountShape, true, nil
}

// removeHistoryReadback re-reads the history: verified when it can no longer be read.
func removeHistoryReadback(c plan.Client, args *pyjson.Object, p *plan.Planned, replies, _ []any) (any, any, any, string, error) {
	id := p.Data.(string)
	exists, count, note, known, err := readHistoryState(c.(ordReader), id)
	if err != nil {
		return nil, nil, nil, "", err
	}
	verdict := "verified"
	switch {
	case !known:
		verdict = "unverified"
	case exists:
		verdict = "mismatch"
	}
	observed := obj("history_id", id, "exists", exists, "record_count", count)
	if !known {
		observed.Set("exists", nil)
		observed.Set("note", note)
	}
	return obj("history_id", id, "acknowledge_data_loss", get(args, "acknowledge_data_loss")), replies[0], observed, verdict, nil
}

// removeHistoryUnapplied is true when the station refused the call and the history still reads.
func removeHistoryUnapplied(c plan.Client, _ *pyjson.Object, p *plan.Planned) (bool, error) {
	r, ok := c.(ordReader)
	if !ok {
		return false, nil
	}
	exists, _, _, err := readHistoryCount(r, p.Data.(string))
	return exists, err
}

// historyNotFound is the station's 404 for a history: an ORD read maps it to a BOX error whose text
// is "HTTP 404 from station" (box.Client.mapError, no Code), measured live on 4.14 after a delete.
func historyNotFound(be *box.Error) bool {
	return be.Code == 404 || (be.Kind == box.KindBox && be.Msg == "HTTP 404 from station")
}
