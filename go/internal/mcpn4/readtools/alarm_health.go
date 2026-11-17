package readtools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_alarm_health (station-audits S4): chattering sources, alarms still out of normal, alarms
// unacknowledged for more than N days and the busiest sources of a window, in one call. Read-only:
// two fixed reads of the alarm: ORD (the records of the window, and the open records), grouped
// here (alarm_health_group.go); never a caller's query, never a write.

// The fixed health queries, their columns read by position.
const (
	// alarmWindowQuery reads each record of the window: its source and when it returned to normal.
	alarmWindowQuery = "select source, normalTime.encodeToString"
	// alarmOpenQuery reads each open record: source, states, class and alarm time.
	alarmOpenQuery = "select source, sourceState.encodeToString, ackState.encodeToString, alarmClass, timestamp.encodeToString"
)

var (
	windowFields = []string{"source", "normal_time"}
	openFields   = []string{"source", "source_state", "ack_state", "alarm_class", "alarm_time"}
	// healthSpanRe is a window: a span back from now (m minutes, h hours, d days).
	healthSpanRe = regexp.MustCompile(`^[0-9]{1,5}[mhd]$`)
)

// Defaults and bounds of n4_alarm_health (the registry schema bounds the integers).
const (
	healthDefaultWindow  = "24h"
	healthDefaultChatter = 10
	healthDefaultDays    = 3
	healthDefaultTop     = 10
	healthMaxRows        = 100000
)

// Messages of n4_alarm_health, as data.
const (
	msgHealthWindow = "window must be a span back from now such as 30m, 24h or 7d, got %q"
	msgHealthCut    = "lists are cut to top=%d entries: %s; counts give the full numbers"
	msgHealthCapped = "the %s read stopped at %d records: its lists and counts cover those records only"
)

// AlarmHealth is n4_alarm_health.
func (t *Tools) AlarmHealth(args *pyjson.Object) (*pyjson.Object, error) {
	window := strArg(args, "window", healthDefaultWindow)
	if _, _, ok := bql.ResolveTime(window); !ok || !healthSpanRe.MatchString(window) {
		return nil, toolError(msgHealthWindow, window)
	}
	chatter, days := intArg(args, "chatter_threshold", healthDefaultChatter), intArg(args, "unacked_days", healthDefaultDays)
	top, timeout := intArg(args, "top", healthDefaultTop), intArg(args, "timeout_s", alarmDefaultTimeout)
	now := bql.Now()
	windowWhere, err := bql.AlarmWhere(bql.AlarmFilter{Since: window})
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	windowRows, windowCapped, ms1, err := t.healthRead(alarmWindowQuery, windowWhere, windowFields, timeout)
	if err != nil {
		return nil, err
	}
	openRows, openCapped, ms2, err := t.healthRead(alarmOpenQuery, bql.AlarmOpenClause, openFields, timeout)
	if err != nil {
		return nil, err
	}
	tallies := tallySources(windowRows)
	chattering, busiest := chatteringSources(tallies, chatter), busiestSources(tallies)
	offnormal, unacked, unreadable := openFindings(openRows, now, days)
	var cut []string
	lists := pyjson.NewObject()
	for _, l := range []struct {
		name  string
		items []any
	}{{"chattering", chattering}, {"offnormal", offnormal}, {"unacked_older", unacked}, {"top_sources", busiest}} {
		if len(l.items) > top {
			cut = append(cut, fmt.Sprintf("%s (%d)", l.name, len(l.items)))
			l.items = l.items[:top]
		}
		lists.Set(l.name, l.items)
	}
	t.recordRead("n4_alarm_health", "", args, len(windowRows)+len(openRows))
	out := obj("window", window, "window_where", windowWhere, "open_where", bql.AlarmOpenClause,
		"chatter_threshold", chatter, "unacked_days", days, "top", top)
	for _, k := range lists.Keys() {
		v, _ := lists.Get(k)
		out.Set(k, v)
	}
	out.Set("counts", obj("records_in_window", len(windowRows), "sources_in_window", len(tallies),
		"chattering", len(chattering), "open_records", len(openRows), "offnormal", len(offnormal),
		"unacked_older", len(unacked), "unreadable_alarm_times", unreadable))
	var notes []any
	if len(cut) > 0 {
		notes = append(notes, fmt.Sprintf(msgHealthCut, top, strings.Join(cut, ", ")))
	}
	for _, c := range []struct {
		name   string
		capped bool
	}{{"window", windowCapped}, {"open", openCapped}} {
		if c.capped {
			notes = append(notes, fmt.Sprintf(msgHealthCapped, c.name, healthMaxRows))
		}
	}
	out.Set("truncated", len(notes) > 0)
	if len(notes) > 0 {
		out.Set("truncated_note", notes)
	}
	out.Set("now", now.UTC().Format(windowSent))
	out.Set("elapsed_ms", ms1+ms2)
	return out, nil
}

// healthRead is one fixed alarm read over where, its rows keyed by position, and whether the
// row cap cut it.
func (t *Tools) healthRead(query, where string, fields []string, timeout int) ([]*pyjson.Object, bool, int, error) {
	columns, rows, capped, ms, err := t.fetchHistory(bql.ComposeAlarmOrd(query, where, ""), healthMaxRows, timeout)
	if err != nil {
		return nil, false, ms, err
	}
	keyed, err := bql.ByPosition(columns, rows, fields)
	if err != nil {
		return nil, false, ms, toolError("%s", err.Error())
	}
	return keyed, capped, ms, nil
}
