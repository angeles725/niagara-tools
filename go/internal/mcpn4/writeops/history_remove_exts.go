package writeops

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The extension read of n4_remove_history: every history extension of the station by slot path,
// enabled and history id (the select histories_audit and n4_read_history use, by position), the
// ones recording into the history kept. Read at the station root: the extension's own path, not
// a caller's argument, is what the write scope is checked against.

const (
	historyExtSelect = "select slotPath, enabled, historyConfig.id from history:HistoryExt"
	historyExtTimout = 60
)

var historyExtFields = []string{"slot", "enabled", "id"}

// historyExt is one extension recording into the history: its station ORD and whether it runs.
type historyExt struct {
	ord     string
	enabled bool
}

// readHistoryExts is the extensions recording into id; a refusal when the read is cut, an
// extension slot is not a plain slot path, or none records into it.
func readHistoryExts(r ordReader, id string) ([]historyExt, error) {
	ord, err := bql.ComposeOrd(bql.StationSlot+"/", historyExtSelect)
	if err != nil {
		return nil, hrRefused(err.Error())
	}
	text, cut, err := r.GetOrd(ord, historyExtTimout, bql.MaxResponseBytes)
	if err != nil {
		return nil, err
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, raw, capped, err := bql.ParseCSV(text, bql.MaxRowsLimit, []int{})
	if err != nil {
		return nil, hrRefused(err.Error())
	}
	rows, err := bql.ByPosition(columns, raw, historyExtFields)
	if err != nil {
		return nil, hrRefused(err.Error())
	}
	if cut || capped {
		return nil, hrRefused(fmt.Sprintf(msgHRCut, len(rows), id))
	}
	var exts []historyExt
	for _, row := range rows {
		if strings.TrimSpace(pyjson.Str(get(row, "id"))) != id {
			continue
		}
		slot := strings.TrimSpace(pyjson.Str(get(row, "slot")))
		extOrd, ok := sourceOrd(slot)
		if !ok {
			return nil, hrRefused(fmt.Sprintf(msgHRSlot, pyjson.StrRepr(slot)))
		}
		enabled := strings.ToLower(strings.TrimSpace(pyjson.Str(get(row, "enabled")))) != "false"
		exts = append(exts, historyExt{ord: extOrd, enabled: enabled})
	}
	if len(exts) == 0 {
		return nil, hrRefused(fmt.Sprintf(msgHRNoExt, id))
	}
	return exts, nil
}
