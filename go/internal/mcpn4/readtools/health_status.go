package readtools

import (
	"math/big"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// healthStatus is one status section: the rows of q counted per status flag, the not-ok ones
// listed up to maxNotOk. Every row is one of ok (no flag), null_only (null alone: no value, not
// a fault) or not ok (any other flag, unknownFlag included), so the three counts sum to total. rows are the query's rows keyed like the inventory's (nil when the
// section failed, with section = {error}); err is only an error that fails the whole call.
func (t *Tools) healthStatus(q healthQuery, base string, maxRows, timeout, maxNotOk int) (section *pyjson.Object, rows []*pyjson.Object, err error) {
	names := append(append([]string{}, inventoryColumns...), bitsColumn)
	if q.enabled {
		names = append(names, enabledColumn)
	}
	columns, raw, cut, _, err := t.query(base, q.query, maxRows, timeout)
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, names)
	}
	if err != nil {
		section, err = t.sectionError(err)
		return section, nil, err
	}
	byFlag, notOk, okCount, nullOnly, notOkCount, disabled := map[string]int{}, []any{}, 0, 0, 0, 0
	for _, row := range rows {
		flags := statusFlags(cellText(row, bitsColumn))
		enabled := enabledCell(cellText(row, enabledColumn))
		if enabled == false {
			disabled++
		}
		if len(flags) == 0 {
			okCount++
			continue
		}
		for _, f := range flags {
			byFlag[f]++
		}
		if len(flags) == 1 && flags[0] == nullFlag { // no value, not a fault
			nullOnly++
			continue
		}
		if notOkCount++; len(notOk) < maxNotOk {
			entry := obj("path", cellText(row, "Slot Path"))
			if q.enabled {
				entry.Set("name", cellText(row, "Name"))
				entry.Set("type", cellText(row, "Type"))
			}
			entry.Set("flags", strs(flags))
			if q.enabled {
				entry.Set("enabled", enabled)
			}
			notOk = append(notOk, entry)
		}
	}
	section = obj("total", len(rows), "ok", okCount, "null_only", nullOnly)
	if q.enabled {
		section.Set("disabled", disabled)
	}
	flagNames := make([]string, 0, len(byFlag))
	for f := range byFlag {
		flagNames = append(flagNames, f)
	}
	sort.Strings(flagNames)
	counts := pyjson.NewObject()
	for _, f := range flagNames {
		counts.Set(f, byFlag[f])
	}
	section.Set("by_flag", counts)
	section.Set("not_ok_count", notOkCount)
	section.Set("not_ok", notOk)
	section.Set("truncated", notOkCount > len(notOk))
	section.Set("rows_truncated", cut)
	return section, rows, nil
}

// statusFlags names the flags of a status.bits cell (a decimal number) through box.StatusFlags;
// a cell that is not a number is the one flag unknownFlag: never an ok status.
func statusFlags(cell string) []string {
	bits, ok := new(big.Int).SetString(strings.TrimSpace(cell), 10)
	if !ok {
		return []string{unknownFlag}
	}
	flags, err := box.StatusFlags(bits.Text(16))
	if err != nil {
		return []string{unknownFlag}
	}
	return flags
}

// enabledCell is an enabled cell as a boolean, nil when the station answered neither.
func enabledCell(cell string) any {
	switch strings.ToLower(strings.TrimSpace(cell)) {
	case "true":
		return true
	case "false":
		return false
	}
	return nil
}

// cellText is the cell of a by-position row as text ("" when absent).
func cellText(row *pyjson.Object, key string) string {
	s, _ := lookup(row, key).(string)
	return s
}
