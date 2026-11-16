package readtools

import (
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_alarm_text_check: the alarm source extensions below base whose texts can never show what
// their author meant (kit rule LC9; rules in alarm_text_rules.go). Read-only: one projected BQL
// select. BQL's from matches subtypes, so every BAlarmSourceExt subclass is read, and a BFormat
// cell is the raw pattern (live 4.15.6.20: "Over amps %parent.out.value% A" comes back as written).

// alarmTextQuery selects slotPath and then the alarmTextSlots in their order, so alarmTextFields
// names each column by position (headers may be localized) and a reordered slice cannot put a
// text in the wrong slot.
var alarmTextQuery = "select slotPath, " + strings.Join(alarmTextSlots, ", ") + " from alarm:AlarmSourceExt"

// alarmTextFields name the query's columns by position.
var alarmTextFields = append([]string{"path"}, alarmTextSlots...)

// AlarmTextCheck is n4_alarm_text_check.
func (t *Tools) AlarmTextCheck(args *pyjson.Object) (*pyjson.Object, error) {
	base := strArg(args, "base", RootOrd)
	maxRows := intArg(args, "max_rows", bql.DefaultMaxRows)
	columns, rows, truncated, elapsed, err := t.query(base, alarmTextQuery, maxRows, intArg(args, "timeout_s", 60))
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, alarmTextFields)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	exts := make([]alarmTextExt, 0, len(keyed))
	for _, row := range keyed {
		e := alarmTextExt{path: slotPathOf(get(row, "path")), texts: map[string]string{}}
		for _, slot := range alarmTextSlots {
			e.texts[slot] = pyjson.Str(get(row, slot))
		}
		exts = append(exts, e)
	}
	sort.SliceStable(exts, func(i, j int) bool { return exts[i].path < exts[j].path })
	findings, parents, empties := []any{}, 0, 0
	for _, e := range exts {
		extRows, p, n := alarmTextFindings(e.path, e.texts)
		findings, parents, empties = append(findings, extRows...), parents+p, empties+n
	}
	t.recordRead("n4_alarm_text_check", "base", args, len(rows))
	out := obj("base", base, "findings", findings, "alarm_extensions", len(exts), "parent_path", parents,
		"empty_tonormal", empties, "summary", fmt.Sprintf("%d alarm source extensions; %d parent-path, %d empty-tonormal [LC9]",
			len(exts), parents, empties), "truncated", truncated, "elapsed_ms", elapsed)
	if truncated {
		out.Set("note", fmt.Sprintf("read cut at %d extensions (max_rows or the byte cap): the counts are lower bounds; "+
			"narrow base or raise max_rows", len(exts)))
	}
	return out, nil
}

// alarmTextExt is one extension: its slot path and its three texts.
type alarmTextExt struct {
	path  string
	texts map[string]string
}
