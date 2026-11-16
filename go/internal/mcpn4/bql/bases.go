package bql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The bases n4_bql_query reads besides a station subtree (retro 2026-11-16 M1): the alarm
// database (alarm:) and one history (history:/<station>/<name>), through the same
// ITableToCsv view. Each base reuses its own composer (ComposeOrd, ComposeAlarmOrd,
// ComposeHistoryOrd); this file only picks one and checks the caller's select.

// stationPrefix is the optional station: hop a caller may put before alarm: or history:.
const stationPrefix = "station:|"

var (
	// tableSelectRe is a select over a table base: the alarm and history tables take no
	// 'from <type>' (validateTableQuery refuses one), so where and order by follow the columns.
	tableSelectRe = regexp.MustCompile(`^(?is)select` + space + `+` + nonSpace + `.*$`)
	// fromRe is a from keyword; quotedRe a quoted literal ('' escapes a quote) masked before it.
	fromRe   = regexp.MustCompile(`(?i)(^|` + space + `)from(` + space + `|$)`)
	quotedRe = regexp.MustCompile(`'(?:[^']|'')*'`)
	// schemeRe is an ORD scheme prefix such as file: or ip:.
	schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

// Messages of the base and table-query checks, as data.
const (
	unknownBaseRule  = "base must be a station slot ORD such as station:|slot:/Drivers, alarm: (the alarm database) or history:/<station>/<history name>, got %s"
	historyBaseRule  = "base %s is not a history ORD: %s"
	tableSelectRule  = "query over the %s base must be 'select <columns> [where ...] [order by ...]'"
	tableFromRule    = "query over the %s base must not have a from clause (the base is the table): use 'select <columns> [where ...] [order by ...]'"
	historyBaseLabel = "history:"
)

// ComposeBaseOrd is the ORD of query over base: a station slot subtree (ComposeOrd, unchanged),
// the alarm database or one history. The base is checked first; a base in another scheme is
// refused naming the three forms, and anything under station: keeps ComposeOrd's refusal.
func ComposeBaseOrd(base, query any) (string, error) {
	b, ok := base.(string)
	if !ok {
		return ComposeOrd(base, query)
	}
	table := strings.TrimPrefix(b, stationPrefix)
	switch {
	case table == AlarmScheme:
		q, err := validateTableQuery(query, AlarmScheme)
		if err != nil {
			return "", err
		}
		return ComposeAlarmOrd(q, "", ""), nil
	case strings.HasPrefix(table, HistoryScheme):
		id := table[len(HistoryScheme):]
		if err := ValidateHistoryId(id); err != nil {
			return "", valueError(fmt.Sprintf(historyBaseRule, pyjson.StrRepr(b), err.Error()))
		}
		q, err := validateTableQuery(query, historyBaseLabel)
		if err != nil {
			return "", err
		}
		return ComposeHistoryOrd(id, "", q)
	case !strings.HasPrefix(b, "station:") && !strings.HasPrefix(b, "slot:") && schemeRe.MatchString(b):
		return "", valueError(fmt.Sprintf(unknownBaseRule, pyjson.StrRepr(b)))
	}
	return ComposeOrd(base, query)
}

// validateTableQuery is ValidateQuery for a table base: the same '|', control and length
// checks, then one select with no from clause (a from inside a quoted literal is data).
func validateTableQuery(query any, base string) (string, error) {
	q, ok := query.(string)
	if !ok {
		return "", valueError(fmt.Sprintf(tableSelectRule, base))
	}
	if err := checkQueryText(Strip(q)); err != nil {
		return "", err
	}
	if q = Strip(q); !tableSelectRe.MatchString(q) {
		return "", valueError(fmt.Sprintf(tableSelectRule, base))
	}
	// Live: history: ignores a from clause and answers the whole history; alarm: answers 404.
	if fromRe.MatchString(quotedRe.ReplaceAllString(q, "''")) {
		return "", valueError(fmt.Sprintf(tableFromRule, base))
	}
	return q, nil
}
