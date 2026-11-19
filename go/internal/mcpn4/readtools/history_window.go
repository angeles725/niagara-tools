package readtools

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The window and status filter of n4_read_history (mcp-n4 delta M2): from/to name the range as
// start/end do, and status keeps only the matching records, filtered by the station so limit,
// truncated and total_records count the matching records only.

// The records query of n4_read_history around its where; with no where it is
// bql.HistoryRecordsQuery exactly.
const (
	historyRecordsSelect = "select timestamp, value, status"
	historyRecordsOrder  = " order by timestamp desc"
)

// Messages of the window and the status filter, as data.
const (
	msgWindowBoth = "give from/to or start/end, not both (from is start, to is end)"
	msgStatusRule = "status must be one of %s, got %s"
)

// historyWindow is the range parameters of args (period, from/to or start/end) and the reply
// keys that echo it. A bound refused by bql.HistoryParams is named as the caller gave it.
func historyWindow(args *pyjson.Object) (params string, echo []string, err error) {
	_, hasFrom := args.Get("from")
	_, hasTo := args.Get("to")
	_, hasStart := args.Get("start")
	_, hasEnd := args.Get("end")
	if (hasFrom || hasTo) && (hasStart || hasEnd) {
		return "", nil, fmt.Errorf("%s", msgWindowBoth)
	}
	startKey, endKey := "start", "end"
	if hasFrom || hasTo {
		startKey, endKey = "from", "to"
	}
	params, err = bql.HistoryParams(strArg(args, "period", ""), strArg(args, startKey, ""), strArg(args, endKey, ""))
	if err != nil && startKey == "from" {
		text := err.Error()
		for _, p := range [][2]string{{"start must", "from must"}, {"end must", "to must"}} {
			if strings.HasPrefix(text, p[0]) {
				text = p[1] + strings.TrimPrefix(text, p[0])
			}
		}
		err = fmt.Errorf("%s", text)
	}
	return params, []string{"period", startKey, endKey}, err
}

// historyStatus is the status filter of args and its BQL where ("" for all).
func historyStatus(args *pyjson.Object) (status, where string, err error) {
	status = strArg(args, "status", "all")
	where, ok := bql.HistoryStatusWhere[status]
	if !ok {
		return "", "", fmt.Errorf(msgStatusRule, strings.Join(bql.HistoryStatusNames, ", "), pyjson.StrRepr(status))
	}
	return status, where, nil
}

// historyRecordsOrd is the ORD of the records of id over params, filtered by where.
func historyRecordsOrd(id, params, where string) (string, error) {
	return bql.ComposeFilteredHistoryOrd(id, params, historyRecordsSelect, where, historyRecordsOrder)
}
