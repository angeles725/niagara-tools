package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// alarmClassesQuery reads the slot path of every alarm class, station-wide: a class sits under
// the AlarmService or an alarm:AlarmClassFolder, and an extension's alarmClass slot holds the
// class's slot name (the last segment).
const alarmClassesQuery = "select slotPath from alarm:AlarmClass"

// stationAlarmClasses is the slot name of each alarm class on the station, in table order; cut
// is true when the table was cut at the byte or row cap. It reads through the BQL route of
// n4_bql_query (ordGetter), from the station root, like the history id check.
func stationAlarmClasses(c plan.Client) (names []string, cut bool, err error) {
	g, ok := c.(ordGetter)
	if !ok {
		return nil, false, plan.ToolError(fmt.Sprintf(msgAlarmClassesUnread, "this client cannot run BQL"))
	}
	ord, err := bql.ComposeOrd(historyIDsBase, alarmClassesQuery)
	if err != nil {
		return nil, false, err
	}
	text, cut, err := g.GetOrd(ord, historyIDsTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, false, plan.ToolError(fmt.Sprintf(msgAlarmClassesUnread, err.Error()))
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, rows, capped, err := bql.ParseCSV(text, bql.MaxRowsLimit, nil)
	if err == nil && len(columns) < 1 {
		err = fmt.Errorf("the answer has no column")
	}
	if err != nil {
		return nil, false, plan.ToolError(fmt.Sprintf(msgAlarmClassesUnread, err.Error()))
	}
	for _, row := range rows { // by position: a localized station translates the header
		v, _ := row.Get(columns[0])
		path := pyjson.Str(v)
		if i := strings.LastIndexByte(path, '/'); i >= 0 && i < len(path)-1 {
			names = append(names, path[i+1:])
		}
	}
	return names, cut || capped, nil
}

// checkAlarmClass refuses a class the station does not hold, listing the ones it does; when the
// table was cut and the class is not in it, the class is left unchecked (unchecked true).
func checkAlarmClass(c plan.Client, class string) (unchecked bool, err error) {
	names, cut, err := stationAlarmClasses(c)
	if err != nil {
		return false, err
	}
	for _, n := range names {
		if n == class {
			return false, nil
		}
	}
	if cut {
		return true, nil
	}
	listed := "none"
	if len(names) > 0 {
		listed = strings.Join(names, ", ")
	}
	return false, plan.ToolError(fmt.Sprintf(msgAlarmClass, pyjson.StrRepr(class), listed))
}
