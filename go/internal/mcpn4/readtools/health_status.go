package readtools

import (
	"math/big"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// statusResult is one status section and what the other sections need of it: the rows keyed
// like the inventory's (nil when the section failed), whether the query was cut, and live, the
// count the station really holds (the rows; for points, the proxy points), known only when the
// query was whole and every row answered.
type statusResult struct {
	section   *pyjson.Object
	rows      []*pyjson.Object
	cut       bool
	live      int
	liveKnown bool
}

// healthStatus is one status section: the rows of q counted per status flag, the not-ok ones
// listed up to maxNotOk. Every row is one of ok (no flag), null_only (null alone: no value, not
// a fault) or not ok (any other flag, unknownFlag included), so the three counts sum to total.
// A failed section is {error} with nil rows; err is only an error that fails the whole call.
func (t *Tools) healthStatus(q healthQuery, base string, maxRows, timeout, maxNotOk int) (statusResult, error) {
	names := append(append([]string{}, inventoryColumns...), bitsColumn)
	switch {
	case q.enabled:
		names = append(names, enabledColumn)
	case q.proxy:
		names = append(names, proxyColumn)
	}
	columns, raw, cut, _, err := t.query(base, q.query, maxRows, timeout)
	var rows []*pyjson.Object
	if err == nil {
		rows, err = bql.ByPosition(columns, raw, names)
	}
	if err != nil {
		section, err := t.sectionError(err)
		return statusResult{section: section}, err
	}
	byFlag, notOk, okCount, nullOnly, notOkCount, disabled := map[string]int{}, []any{}, 0, 0, 0, 0
	proxy, proxyUnknown := 0, 0
	for _, row := range rows {
		flags := statusFlags(cellText(row, bitsColumn))
		enabled := enabledCell(cellText(row, enabledColumn))
		if enabled == false {
			disabled++
		}
		switch ext := strings.TrimSpace(cellText(row, proxyColumn)); {
		case !q.proxy:
		case ext == "":
			proxyUnknown++
		case ext != nullProxyExt:
			proxy++
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
			notOk = append(notOk, notOkEntry(q, row, flags, enabled))
		}
	}
	section := obj("total", len(rows), "ok", okCount, "null_only", nullOnly)
	res := statusResult{section: section, rows: rows, cut: cut, live: len(rows), liveKnown: !cut}
	if q.enabled {
		section.Set("disabled", disabled)
	}
	if q.proxy {
		section.Set("proxy", proxy)
		if proxyUnknown > 0 {
			section.Set("proxy_unknown", proxyUnknown)
		}
		res.live, res.liveKnown = proxy, !cut && proxyUnknown == 0
	}
	section.Set("by_flag", flagCounts(byFlag))
	section.Set("not_ok_count", notOkCount)
	section.Set("not_ok", notOk)
	section.Set("truncated", notOkCount > len(notOk))
	section.Set("rows_truncated", cut)
	return res, nil
}

// notOkEntry is one entry of a section's not_ok list.
func notOkEntry(q healthQuery, row *pyjson.Object, flags []string, enabled any) *pyjson.Object {
	entry := obj("path", cellText(row, "Slot Path"))
	if q.enabled {
		entry.Set("name", cellText(row, "Name"))
		entry.Set("type", cellText(row, "Type"))
	}
	entry.Set("flags", strs(flags))
	if q.enabled {
		entry.Set("enabled", enabled)
	}
	return entry
}

// flagCounts is the count per flag, by flag name.
func flagCounts(byFlag map[string]int) *pyjson.Object {
	names := make([]string, 0, len(byFlag))
	for f := range byFlag {
		names = append(names, f)
	}
	sort.Strings(names)
	counts := pyjson.NewObject()
	for _, f := range names {
		counts.Set(f, byFlag[f])
	}
	return counts
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
