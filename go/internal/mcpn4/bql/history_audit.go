package bql

import (
	"strconv"
	"strings"
	"time"
)

// The station reads of n4_histories_audit (station-audits S1) and the encoded history settings
// they answer, each as its encodeToString text: BQL lists every public no-argument method of a
// value as a field (ScriptFields.introspect), so encodeToString reads the machine form, not
// the localized display text.

// HistoriesAuditExtQuery reads each history extension, by position: slot path, type, enabled,
// history id, capacity (BCapacity "1:1000"), full policy ("roll" or "stop"), collection
// interval (BCollectionInterval "false:300000", "true:0" for COV) and the last record time.
const HistoriesAuditExtQuery = "select slotPath, type, enabled, historyConfig.id, " +
	"historyConfig.capacity.encodeToString, historyConfig.fullPolicy.encodeToString, " +
	"historyConfig.interval.encodeToString, lastRecord.timestamp.encodeToString from history:HistoryExt"

// HistoriesAuditPointQuery reads the slot path of each control point.
const HistoriesAuditPointQuery = "select slotPath from control:ControlPoint"

// HistoriesAuditFlatPointQuery reads each control point's slot path, status bits (the status as a number,
// which no locale translates) and facets string, by position: n4_histories_audit flat_scope and flat_down.
const HistoriesAuditFlatPointQuery = "select slotPath, status.bits, facets from control:ControlPoint"

// HistoryTypeQuery reads each history extension's slot path, history id, type and collection
// interval (encodeToString, as in HistoriesAuditExtQuery), by position: n4_read_history
// history_type (mcp-n4 delta M2).
const HistoryTypeQuery = "select slotPath, historyConfig.id, type, historyConfig.interval.encodeToString " +
	"from history:HistoryExt"

// HistoryStatusNames are the n4_read_history status filters, in order; HistoryStatusWhere is the
// BQL where of each ("all" has none). Live-checked on 4.14.0.162 (2026-11-19): status.isOk,
// status.fault and status.null select records; status.ok matches nothing and returns zero rows
// with no error, so ok / not-ok use isOk. status.null is also live-checked on 4.15.6.20.
var (
	HistoryStatusNames = []string{"all", "ok", "not-ok", "fault", "down", "stale", "alarm", "overridden",
		"disabled", "null"}
	HistoryStatusWhere = map[string]string{"all": "", "ok": "status.isOk = true", "not-ok": "status.isOk = false",
		"fault": "status.fault = true", "down": "status.down = true", "stale": "status.stale = true",
		"alarm": "status.alarm = true", "overridden": "status.overridden = true",
		"disabled": "status.disabled = true", "null": "status.null = true"}
)

// The BCapacity restrictions (BCapacity.RESTRICT_NONE, RESTRICT_RECORD_COUNT, RESTRICT_STORAGE_SIZE).
const (
	CapacityUnlimited = 0
	CapacityRecords   = 1
	CapacityBytes     = 2
)

// Capacity is a decoded BCapacity: what it restricts and the most it holds (0 when unlimited).
type Capacity struct {
	RestrictBy int
	Max        int64
}

// ParseCapacity reads BCapacity.encodeToString, "<restrictBy>:<max>"; ok is false for any other
// text (a localized display such as "1000 records" included).
func ParseCapacity(text string) (Capacity, bool) {
	by, max, found := strings.Cut(strings.TrimSpace(text), ":")
	r, err1 := strconv.Atoi(by)
	m, err2 := strconv.ParseInt(max, 10, 64)
	if !found || err1 != nil || err2 != nil || r < CapacityUnlimited || r > CapacityBytes || m < 0 {
		return Capacity{}, false
	}
	return Capacity{RestrictBy: r, Max: m}, true
}

// ParseCollectionInterval reads BCollectionInterval.encodeToString, "<irregular>:<ms>": an
// irregular (COV) history, or the interval of a regular one (never zero).
func ParseCollectionInterval(text string) (irregular bool, every time.Duration, ok bool) {
	flag, ms, found := strings.Cut(strings.TrimSpace(text), ":")
	n, err := strconv.ParseInt(ms, 10, 64)
	switch {
	case !found || err != nil || n < 0:
		return false, 0, false
	case flag == "true":
		return true, 0, true
	case flag == "false" && n > 0:
		return false, time.Duration(n) * time.Millisecond, true
	}
	return false, 0, false
}

// ParseSpan reads a span such as 30m, 24h or 7d (the relative bounds' form; never 0).
func ParseSpan(text string) (time.Duration, bool) {
	m := relTimeRe.FindStringSubmatch(text)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[1])
	return time.Duration(n) * relUnits[m[2]], n > 0
}

// ParseAbsTime reads BAbsTime.encodeToString; ok is false for the null time and any other text.
func ParseAbsTime(text string) (time.Time, bool) {
	if IsoOrNull(text) == nil || !absTimeRe.MatchString(text) {
		return time.Time{}, false
	}
	t, err := time.Parse(absTimeLayout, text)
	return t, err == nil
}
