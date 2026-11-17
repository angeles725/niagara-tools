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
	// HistorySummaryQuery reads every record of the range with machine-encoded time and value
	// (encodeToString: ISO-8601 with the station's offset, 21.0 not a localized "21,0").
	HistorySummaryQuery = "select timestamp.encodeToString, value.encodeToString, status order by timestamp desc"
	// HistoryAggregateQuery is the whole-range min and max as the station renders them, without
	// the records whose status is null (no value; the summary's rule, retro 2026-11-16 row M16b).
	// The where form is live-checked on 4.15.6.20: status.null = false (status.isNull() is 400).
	HistoryAggregateQuery = "select min(value), max(value) where status.null = false"
	HistoryCountQuery     = "select count(*)"
	HistoryScheme         = "history:"
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
	timeRule   = "%s must be a BAbsTime such as 2026-11-15T00:00:00.000-06:00 or 2026-11-15T06:00:00Z, or a span back from now such as 30m, 24h or 7d, got %q"
	rangeRule  = "give either period or start/end, not both"
	orderRule  = "start must not be later than end"
	nameRule   = "history_id must be /<station>/<history name>, or a bare history name of the connected station " +
		"(a Niagara slot name: a letter, then letters, digits or '_', other characters $-escaped; at most %d characters), got %s"
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

// ResolveHistoryId is id when it holds a '/', else the bare history name id of station as
// /<station>/<id>; a bare name that is not a slot name is refused naming the caller's text.
func ResolveHistoryId(id, station string) (string, error) {
	if id == "" || strings.Contains(id, "/") {
		return id, ValidateHistoryId(id)
	}
	if !slotNameRe.MatchString(id) || len(id) > MaxHistoryNameLength {
		return "", valueError(fmt.Sprintf(nameRule, MaxHistoryNameLength, pyjson.StrRepr(id)))
	}
	full := "/" + station + "/" + id
	return full, ValidateHistoryId(full)
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
	var bounds [2]int64
	for i, b := range [][2]string{{"start", start}, {"end", end}} {
		if b[1] == "" {
			continue
		}
		at, ms, ok := ResolveTime(b[1])
		if !ok {
			return "", valueError(fmt.Sprintf(timeRule, b[0], b[1]))
		}
		bounds[i] = ms
		params = append(params, b[0]+"="+at)
	}
	if start != "" && end != "" && bounds[0] > bounds[1] {
		return "", valueError(orderRule)
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
