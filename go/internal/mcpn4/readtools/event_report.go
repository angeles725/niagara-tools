package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_event_report (retro 2026-11-16 P3, "a cuanto llego" took 6 steps): the alarm events of a
// source, newest first, each with its alarm database times and the summary (count, min, max and
// when, first, last) of the source's history over the event window. Read-only: the fixed alarm
// queries of n4_read_alarms, the extension lookup and the summary read of n4_read_history.

// eventFields name the event columns by position (bql.AlarmEventQuery).
var eventFields = []string{"alarm_time", "uuid", "source", "source_state", "ack_state", "alarm_class",
	"transition", "normal_time", "ack_time", "user"}

// Defaults of n4_event_report (the registry schema bounds max_events and margin_minutes).
const (
	eventDefaultMax    = 10
	eventDefaultMargin = 15
	eventSummaryRows   = 100000
)

// Messages of n4_event_report, as data.
const (
	msgEventOneSource = "give exactly one of source (a component ORD such as station:|slot:/Folder/Point/AlarmExt) " +
		"or source_glob (a like pattern such as *slot:/Folder/*)"
)

// EventReport is n4_event_report.
func (t *Tools) EventReport(args *pyjson.Object) (*pyjson.Object, error) {
	source, glob := strArg(args, "source", ""), strArg(args, "source_glob", "")
	if (source == "") == (glob == "") {
		return nil, toolError("%s", msgEventOneSource)
	}
	where, err := bql.AlarmWhere(bql.AlarmFilter{Source: source, SourceGlob: glob,
		Since: strArg(args, "since", ""), Until: strArg(args, "until", "")})
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	maxEvents, margin := intArg(args, "max_events", eventDefaultMax), intArg(args, "margin_minutes", eventDefaultMargin)
	historyID := strArg(args, "history_id", "")
	if historyID != "" {
		if historyID, err = bql.ResolveHistoryId(historyID, t.Ctx.Session.StationName); err != nil {
			return nil, toolError("%s", err.Error())
		}
	}
	timeout := intArg(args, "timeout_s", alarmDefaultTimeout)
	ord := bql.ComposeAlarmOrd(bql.AlarmEventQuery, where, bql.AlarmOrder)
	columns, rows, truncated, elapsed, err := t.fetchHistory(ord, maxEvents, timeout)
	if err != nil {
		return nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, eventFields)
	if err != nil {
		return nil, toolError("%s", err.Error())
	}
	total, note, ms := t.countRows(bql.ComposeAlarmOrd(bql.AlarmCountQuery, where, ""), timeout)
	elapsed += ms
	w := &eventWindows{t: t, historyID: historyID, margin: margin, timeout: timeout, found: map[string]*historyFound{}}
	evs := groupEvents(keyed)
	out := make([]any, len(evs))
	for i, ev := range evs {
		out[i] = w.report(ev)
	}
	t.recordRead("n4_event_report", "source", args, len(rows))
	reply := obj("where", where, "margin_minutes", margin, "events", out, "event_count", len(evs),
		"order", "newest_first", "truncated", truncated, "total_events", total)
	if note != "" {
		reply.Set("total_events_note", note)
	}
	reply.Set("elapsed_ms", elapsed+w.elapsed)
	return reply, nil
}

// event is one alarm event: its newest record and how many records carry its uuid.
type event struct {
	row     *pyjson.Object
	records int
}

// groupEvents are rows grouped by uuid in reading order. The database keeps one record per
// event, so records is 1 unless the station answers a uuid twice.
func groupEvents(rows []*pyjson.Object) []*event {
	var evs []*event
	byUUID := map[string]*event{}
	for _, row := range rows {
		uuid := cell(row, "uuid")
		if ev, ok := byUUID[uuid]; ok {
			ev.records++
			continue
		}
		ev := &event{row: row, records: 1}
		byUUID[uuid] = ev
		evs = append(evs, ev)
	}
	return evs
}

// eventFacts are the record's identity, states and times as the report shows them.
func eventFacts(ev *event) *pyjson.Object {
	o := pyjson.NewObject()
	for _, k := range []string{"uuid", "source", "alarm_class", "transition", "source_state", "ack_state"} {
		o.Set(k, cell(ev.row, k))
	}
	for _, k := range []string{"alarm_time", "normal_time", "ack_time"} {
		o.Set(k, bql.IsoOrNull(cell(ev.row, k)))
	}
	var ackedBy any // the record's user is "Unknown User" until someone acknowledges it
	if ack, _ := o.Get("ack_time"); ack != nil {
		ackedBy = cell(ev.row, "user")
	}
	o.Set("acked_by", ackedBy)
	o.Set("records", ev.records)
	return o
}

// windowNote is why an event has no window.
func windowNote(err error) string { return fmt.Sprintf("no event window: %s", err.Error()) }
