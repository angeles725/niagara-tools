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
	// schemeRe is an ORD scheme prefix such as file: or ip:.
	schemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)
)

// Messages of the base and table-query checks, as data.
const (
	unknownBaseRule  = "base must be a station slot ORD such as station:|slot:/Drivers, alarm: (the alarm database) or history:/<station>/<history name>, got %s"
	historyBaseRule  = "base %s is not a history ORD: %s"
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
