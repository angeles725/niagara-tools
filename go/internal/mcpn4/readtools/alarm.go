package readtools

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_read_alarms: the station's alarm records, open ones by default, newest first, narrowed by
// validated filters. Read-only: fixed queries over the alarm: ORD (one BQL GET for the records
// and one select count(*)), never a caller's query, never a write.

// alarmFields name the record columns by position (a localized station translates headers).
var alarmFields = []string{"timestamp", "uuid", "source", "source_state", "ack_state", "alarm_class",
	"priority", "msg_text"}

// Defaults of n4_read_alarms (the registry schema caps limit at 1000).
const (
	alarmDefaultLimit   = 50
	alarmDefaultTimeout = 60
)

// Messages of n4_read_alarms, as data.
const (
	msgAlarmCountFailed = "the alarm count (select count(*)) failed: %s"
	msgAlarmCountShape  = "the alarm count (select count(*)) answered no whole number"
)

// ReadAlarms is n4_read_alarms.
func (t *Tools) ReadAlarms(args *pyjson.Object) (*pyjson.Object, error) {
	open := true
	if v, ok := args.Get("open_only"); ok {
		open = pyjson.Truthy(v)
	}
	where, err := bql.AlarmWhere(bql.AlarmFilter{Source: strArg(args, "source", ""),
		AlarmClass: strArg(args, "alarm_class", ""), SourceState: strArg(args, "source_state", ""),
		AckState: strArg(args, "ack_state", ""), Since: strArg(args, "since", ""), Until: strArg(args, "until", ""),
		OpenOnly: open})
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	limit, timeout := intArg(args, "limit", alarmDefaultLimit), intArg(args, "timeout_s", alarmDefaultTimeout)
	ord := bql.ComposeAlarmOrd(bql.AlarmRecordsQuery, where, bql.AlarmOrder)
	columns, rows, truncated, ms, err := t.fetchHistory(ord, limit, timeout)
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, alarmFields)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	total, note, countMs := t.countRows(bql.ComposeAlarmOrd(bql.AlarmCountQuery, where, ""), timeout)
	t.recordRead("n4_read_alarms", "source", args, len(rows))
	out := obj("where", where, "columns", strs(columns), "rows", list(keyed), "row_count", len(rows),
		"order", "newest_first", "truncated", truncated, "total", total)
	if note != "" {
		out.Set("total_note", note)
	}
	out.Set("elapsed_ms", ms+countMs)
	return out, nil
}

// countRows is the whole number a select count(*) ORD answers, or nil and a note saying why not.
func (t *Tools) countRows(ord string, timeout int) (any, string, int) {
	_, rows, _, elapsed, err := t.fetchHistory(ord, 1, timeout)
	if err != nil {
		return nil, fmt.Sprintf(msgAlarmCountFailed, err.Error()), elapsed
	}
	if len(rows) == 1 && len(rows[0].Keys()) == 1 {
		v, _ := rows[0].Get(rows[0].Keys()[0])
		if n, err := strconv.Atoi(strings.TrimSpace(pyjson.Str(v))); err == nil && n >= 0 {
			return n, "", elapsed
		}
	}
	return nil, msgAlarmCountShape, elapsed
}
