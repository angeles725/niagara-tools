package writeops

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// historyIDsQuery reads every history extension's slot path and history id
// ("/<stationName>/<historyName>", history-rt BHistoryExt: BHistoryId.make(stationName,
// historyName)), station-wide, in one BQL read.
const historyIDsQuery = "select slotPath, historyConfig.id from history:HistoryExt"

// historyIDsBase is where that read starts: the station root.
const historyIDsBase = "station:|slot:/"

// historyIDsTimeout bounds that read, in seconds.
const historyIDsTimeout = 60

// ordGetter is a client that can GET an ORD (box.Client.GetOrd), the BQL route of n4_bql_query.
type ordGetter interface {
	GetOrd(ord string, timeout float64, maxBytes int) (string, bool, error)
}

// stationHistoryIDs maps each history id the station's extensions hold to that extension's
// slot path; cut is true when the table was cut at the byte or row cap.
func stationHistoryIDs(c plan.Client) (ids map[string]string, cut bool, err error) {
	g, ok := c.(ordGetter)
	if !ok {
		return nil, false, plan.ToolError(fmt.Sprintf(msgHistoryIDsUnread, "this client cannot run BQL"))
	}
	ord, err := bql.ComposeOrd(historyIDsBase, historyIDsQuery)
	if err != nil {
		return nil, false, err
	}
	text, cut, err := g.GetOrd(ord, historyIDsTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, false, plan.ToolError(fmt.Sprintf(msgHistoryIDsUnread, err.Error()))
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, rows, capped, err := bql.ParseCSV(text, bql.MaxRowsLimit, nil)
	if err == nil && len(columns) < 2 {
		err = fmt.Errorf("the answer has %d columns, not 2", len(columns))
	}
	if err != nil {
		return nil, false, plan.ToolError(fmt.Sprintf(msgHistoryIDsUnread, err.Error()))
	}
	ids = map[string]string{}
	for _, row := range rows { // by position: a localized station translates the headers
		slot, _ := row.Get(columns[0])
		id, _ := row.Get(columns[1])
		if s, ok := id.(string); ok && s != "" {
			ids[s] = pyjson.Str(slot)
		}
	}
	return ids, cut || capped, nil
}

// checkStationIDs refuses the first planned extension whose history id an existing extension
// already holds; it reads the station only when some id was predicted.
func checkStationIDs(c plan.Client, specs []historySpec) (cut bool, err error) {
	predicted := false
	for _, s := range specs {
		predicted = predicted || s.id != ""
	}
	if !predicted {
		return false, nil
	}
	ids, cut, err := stationHistoryIDs(c)
	if err != nil {
		return false, err
	}
	for _, s := range specs {
		if slot, taken := ids[s.id]; taken && s.id != "" {
			return false, plan.ToolError(fmt.Sprintf(msgHistoryIDTaken, pyjson.StrRepr(s.id), s.ord, slot))
		}
	}
	return cut, nil
}
