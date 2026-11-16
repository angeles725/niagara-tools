package bql

import (
	"fmt"
	"regexp"
	"strings"
)

// Alarm reads: the alarm: scheme ORD of the station's alarm database, with a where clause built
// only from validated filter values. ComposeOrd stays the only composer of station: BQL ORDs;
// this one never takes a caller's query.

// The fixed alarm queries: the records newest first (columns read by position), and their count.
const (
	AlarmScheme       = "alarm:"
	AlarmRecordsQuery = "select timestamp, uuid, source, sourceState, ackState, alarmClass, priority, alarmData.msgText, " +
		"timestamp.encodeToString, normalTime.encodeToString, ackTime.encodeToString"
	AlarmCountQuery = "select count(*)"
	// AlarmStatesQuery reads each matching record's state tags, counted per state.
	AlarmStatesQuery = "select sourceState.encodeToString, ackState.encodeToString"
	AlarmOrder       = " order by timestamp desc"
	// AlarmOpenClause is the default filter: an alarm still unacknowledged or not back to normal.
	AlarmOpenClause = "(ackState != 'acked' or sourceState != 'normal')"
	// alarmSourceClause matches a record source naming the component at a slot path or below
	// it, in both forms a station stores (slot:/a and local:|station:|slot:/a). It holds no
	// '|': the ORD separator would cut the bql part (HTTP 400 live), so '*' stands for it.
	alarmSourceClause = "(source like 'slot:%[1]s' or source like 'slot:%[1]s/*' or " +
		"source like 'local:*slot:%[1]s' or source like 'local:*slot:%[1]s/*')"
)

// alarmGlobRe is a source_glob: ORD characters and '*' only; no quote, space or '|' (the ORD
// separator would cut the bql part, so '*' stands for it).
// No '$' either: the station would decode $xx into any character ('*' matches an escaped one).
var alarmGlobRe = regexp.MustCompile(`^[A-Za-z0-9_:/.*-]{1,400}$`)

// The BAlarmRecord enum tags a filter may name (BSourceState, BAckState).
var (
	AlarmSourceStates = []string{"normal", "offnormal", "fault", "alert"}
	AlarmAckStates    = []string{"acked", "unacked", "ackPending"}
)

// Messages of the alarm filter checks, as data.
const (
	alarmEnumRule   = "%s must be one of %s, got %q"
	alarmClassRule  = "alarm_class must be an alarm class name (a letter, then letters, digits or '_', other characters $-escaped), got %q"
	alarmSourceRule = "source must be a component ORD such as station:|slot:/Drivers/Net/Device (each segment a slot name), got %q"
	alarmGlobRule   = "source_glob must be a like pattern of ORD characters and '*' (no quote, space, '$' or '|': '*' stands for '|' and for an escaped character), e.g. *slot:/Drivers/Net/*, got %q"
	alarmTimeRule   = "%s must be a BAbsTime such as 2026-11-15T00:00:00.000-06:00 or 2026-11-15T06:00:00Z, or a span back from now such as 30m, 24h or 7d, got %q"
	alarmRangeRule  = "since must not be later than until"
)

// AlarmFilter is what a caller may narrow the alarm read with; "" is no filter. OpenOnly applies
// AlarmOpenClause when neither state filter is given.
type AlarmFilter struct {
	Source, SourceGlob, AlarmClass, SourceState, AckState, Since, Until string
	OpenOnly                                                            bool
}

// AlarmWhere is the where clause of f ("" for none): every value checked against its grammar,
// so no quote, wildcard or keyword of the caller ever reaches the query.
func AlarmWhere(f AlarmFilter) (string, error) {
	var conds []string
	if f.Source != "" {
		path, err := alarmSourcePath(f.Source)
		if err != nil {
			return "", err
		}
		if err := checkEscapesShown("source", path, f.Source, literalForbidden); err != nil {
			return "", err
		}
		conds = append(conds, fmt.Sprintf(alarmSourceClause, path))
	}
	if f.SourceGlob != "" {
		if !alarmGlobRe.MatchString(f.SourceGlob) {
			return "", valueError(fmt.Sprintf(alarmGlobRule, f.SourceGlob))
		}
		conds = append(conds, "source like '"+f.SourceGlob+"'")
	}
	if f.AlarmClass != "" {
		if !slotNameRe.MatchString(f.AlarmClass) {
			return "", valueError(fmt.Sprintf(alarmClassRule, f.AlarmClass))
		}
		if err := checkEscapes("alarm_class", f.AlarmClass, literalForbidden); err != nil {
			return "", err
		}
		conds = append(conds, "alarmClass = '"+f.AlarmClass+"'")
	}
	for _, e := range []struct {
		arg, column, value string
		tags               []string
	}{
		{"source_state", "sourceState", f.SourceState, AlarmSourceStates},
		{"ack_state", "ackState", f.AckState, AlarmAckStates}} {
		if e.value == "" {
			continue
		}
		if !contains(e.tags, e.value) {
			return "", valueError(fmt.Sprintf(alarmEnumRule, e.arg, strings.Join(e.tags, ", "), e.value))
		}
		conds = append(conds, e.column+" = '"+e.value+"'")
	}
	if f.OpenOnly && f.SourceState == "" && f.AckState == "" {
		conds = append(conds, AlarmOpenClause)
	}
	var since, until int64
	for _, b := range []struct {
		arg, value, op string
		ms             *int64
	}{{"since", f.Since, ">=", &since}, {"until", f.Until, "<=", &until}} {
		if b.value == "" {
			continue
		}
		ms, err := absTimeMillis(b.arg, b.value)
		if err != nil {
			return "", err
		}
		*b.ms = ms
		conds = append(conds, fmt.Sprintf("timestamp.millis %s %d", b.op, ms))
	}
	if f.Since != "" && f.Until != "" && since > until {
		return "", valueError(alarmRangeRule)
	}
	return strings.Join(conds, " and "), nil
}

// ComposeAlarmOrd is the ORD of query (one of the fixed alarm queries) over the alarm database,
// with where and the order suffix appended.
func ComposeAlarmOrd(query, where, order string) string {
	if where != "" {
		query += " where " + where
	}
	return AlarmScheme + "|bql:" + query + order + "|" + View
}

// alarmSourcePath is the slot path of a source filter: station:|slot:/a/b or slot:/a/b, each
// segment a slot name.
func alarmSourcePath(source string) (string, error) {
	path := strings.TrimPrefix(strings.TrimPrefix(source, "station:|"), "slot:")
	if path == source || !strings.HasPrefix(path, "/") || path == "/" {
		return "", valueError(fmt.Sprintf(alarmSourceRule, source))
	}
	for _, seg := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if !slotNameRe.MatchString(seg) {
			return "", valueError(fmt.Sprintf(alarmSourceRule, source))
		}
	}
	return path, nil
}

// absTimeMillis is the epoch milliseconds of a BAbsTime text or a relative span (ResolveTime).
func absTimeMillis(arg, value string) (int64, error) {
	_, ms, ok := ResolveTime(value)
	if !ok {
		return 0, valueError(fmt.Sprintf(alarmTimeRule, arg, value))
	}
	return ms, nil
}
