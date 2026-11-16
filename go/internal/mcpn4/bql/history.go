package bql

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// History reads: the history: scheme ORD of one history id, with an optional time range, and
// the fixed queries it carries. ComposeOrd stays the only composer of station: BQL ORDs; this
// one never takes a caller's query.

// The fixed history queries: the records newest first, and their count.
const (
	HistoryRecordsQuery = "select timestamp, value, status order by timestamp desc"
	HistoryCountQuery   = "select count(*)"
	HistoryScheme       = "history:"
)

// HistoryPeriods are the period ids BDynamicTimeRange.make accepts (bql-rt BDynamicTimeRange
// byId; HistoryQuery.parseTimeRange). "timeRange" is left out: it means start/end.
var HistoryPeriods = []string{"today", "last24Hours", "yesterday", "weekToDate", "lastWeek",
	"last7Days", "monthToDate", "lastMonth", "yearToDate", "lastYear"}

// MaxHistoryNameLength is BHistoryId.MAX_NAME_LENGTH.
const MaxHistoryNameLength = 200

var (
	// slotNameRe is SlotPath.isValidName (EscUtil.SlotEsc): a letter, then letters, digits or
	// '_', each character possibly a $xx or $uxxxx escape.
	slotNameRe = regexp.MustCompile(`^(?:[A-Za-z]|\$[0-9A-Fa-f]{2}|\$u[0-9A-Fa-f]{4})(?:[A-Za-z0-9_]|\$[0-9A-Fa-f]{2}|\$u[0-9A-Fa-f]{4})*$`)
	// absTimeRe is the form BAbsTime.decodeFromString reads: yyyy-mm-ddThh:mm:ss[.fff](Z|+hh:mm|-hh:mm).
	absTimeRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?(?:Z|[+-]\d{2}:\d{2})$`)
)

// Messages of the history ORD checks.
const (
	historyIdRule = "history_id must be /<station>/<history name>: two segments, each a Niagara slot " +
		"name (a letter, then letters, digits or '_', other characters $-escaped as $20 or $u00e9), " +
		"the name at most %d characters, got %s"
	periodRule = "period must be one of %s, got %q"
	timeRule   = "%s must be a BAbsTime such as 2026-11-15T00:00:00.000-06:00 or 2026-11-15T06:00:00Z, got %q"
	rangeRule  = "give either period or start/end, not both"
)

// ValidateHistoryId is nil when id is /<device>/<name> as BHistoryId.make parses it, both
// segments valid slot names; ids with '|', '?', ';', spaces or controls never pass.
func ValidateHistoryId(id string) error {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || parts[0] != "" || !slotNameRe.MatchString(parts[1]) ||
		!slotNameRe.MatchString(parts[2]) || len(parts[2]) > MaxHistoryNameLength {
		return valueError(fmt.Sprintf(historyIdRule, MaxHistoryNameLength, pyjson.StrRepr(id)))
	}
	return nil
}

// HistoryParams is the HistoryQuery parameter string of a period or a start/end range ("" for
// the whole history): period=<id>, or start=<t>;end=<t> with either bound optional.
func HistoryParams(period, start, end string) (string, error) {
	if period != "" {
		if start != "" || end != "" {
			return "", valueError(rangeRule)
		}
		for _, p := range HistoryPeriods {
			if p == period {
				return "period=" + period, nil
			}
		}
		return "", valueError(fmt.Sprintf(periodRule, strings.Join(HistoryPeriods, ", "), period))
	}
	var params []string
	for _, b := range [][2]string{{"start", start}, {"end", end}} {
		if b[1] == "" {
			continue
		}
		if !absTimeRe.MatchString(b[1]) {
			return "", valueError(fmt.Sprintf(timeRule, b[0], b[1]))
		}
		params = append(params, b[0]+"="+b[1])
	}
	return strings.Join(params, ";"), nil
}

// ComposeHistoryOrd is the ORD of query over the history id, its parameters after '?'.
func ComposeHistoryOrd(id, params, query string) (string, error) {
	if err := ValidateHistoryId(id); err != nil {
		return "", err
	}
	ord := HistoryScheme + id
	if params != "" {
		ord += "?" + params
	}
	return ord + "|bql:" + query + "|" + View, nil
}
