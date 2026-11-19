package readtools

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_history history_type=true (mcp-n4 delta M2): whether the history records on change of
// value (cov) or every interval (with its period), from the collection interval of the history
// extension that records into it: one station-wide read of the history extensions.

// Notes of history_type, as data.
const (
	msgTypeNone = "no history extension records into %s (an imported or archived history, or one " +
		"whose extension was removed)%s: its type is not known"
	msgTypeCut     = "; the extension read stopped at %d rows"
	msgTypeFailed  = "the history extensions could not be read (%s): %s"
	msgTypeUnknown = "the extension %s answered its collection interval in no known form (%s)"
)

// historyType is the history_type of id, or nil and a note saying why there is none, and the
// read's latency. An AuthError fails the call (the session latches on it).
func (t *Tools) historyType(id string, timeout int) (any, string, int, error) {
	columns, rows, cut, ms, err := t.query(RootOrd, bql.HistoryTypeQuery, bql.DefaultMaxRows, timeout)
	var keyed []*pyjson.Object
	if err == nil {
		keyed, err = bql.ByPosition(columns, rows, []string{"slot", "id", "type", "interval"})
	}
	if err != nil {
		var be *box.Error
		if errors.As(err, &be) && be.IsAuth() {
			return nil, "", ms, err
		}
		return nil, fmt.Sprintf(msgTypeFailed, bql.HistoryTypeQuery, err.Error()), ms, nil
	}
	var found *pyjson.Object
	var others []any
	for _, row := range keyed {
		if strings.TrimSpace(cell(row, "id")) != id {
			continue
		}
		if found != nil {
			others = append(others, cell(row, "slot"))
			continue
		}
		e := histExt{path: cell(row, "slot"), typ: cell(row, "type"), interval: cell(row, "interval")}
		cov, every, ok := isCov(e)
		if !ok {
			return nil, fmt.Sprintf(msgTypeUnknown, pyjson.StrRepr(e.path), pyjson.StrRepr(e.interval)), ms, nil
		}
		found = obj("kind", "cov", "interval", nil, "interval_ms", nil, "ext", e.path, "ext_type", e.typ)
		if !cov {
			found.Set("kind", "interval")
			found.Set("interval", spanText(every))
			found.Set("interval_ms", int(every.Milliseconds()))
		}
	}
	if found == nil {
		more := ""
		if cut {
			more = fmt.Sprintf(msgTypeCut, len(keyed))
		}
		return nil, fmt.Sprintf(msgTypeNone, id, more), ms, nil
	}
	if len(others) > 0 {
		found.Set("other_exts", others)
	}
	return found, "", ms, nil
}
