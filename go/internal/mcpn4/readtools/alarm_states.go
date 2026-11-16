package readtools

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The state counts of n4_read_alarms (retro 2026-11-16 M2): every record the filters match,
// counted per source state and per ack state from one read of their state tags, so the counts
// cover the whole match whatever limit cut the rows.

// alarmStateRows bounds the state read (about 20 bytes a record).
const alarmStateRows = 200000

// msgAlarmStatesFailed is why state_counts is null.
const msgAlarmStatesFailed = "the state counts (%s) failed: %s"

// alarmStateCounts is {source_state, ack_state, records_counted, complete} over the records
// where matches, or nil and a note saying why not. Every known tag is listed, zero included;
// a tag the station adds is counted after them.
func (t *Tools) alarmStateCounts(where string, timeout int) (any, string, int) {
	_, rows, capped, elapsed, err := t.fetchHistory(bql.ComposeAlarmOrd(bql.AlarmStatesQuery, where, ""), alarmStateRows, timeout)
	if err == nil && len(rows) > 0 && len(rows[0].Keys()) < 2 {
		err = fmt.Errorf("the answer has %d column(s), expected 2", len(rows[0].Keys()))
	}
	if err != nil {
		return nil, fmt.Sprintf(msgAlarmStatesFailed, bql.AlarmStatesQuery, err.Error()), elapsed
	}
	source, ack := tagCounts(bql.AlarmSourceStates), tagCounts(bql.AlarmAckStates)
	for _, row := range rows {
		keys := row.Keys()
		for i, counts := range []*pyjson.Object{source, ack} {
			v, _ := row.Get(keys[i])
			tag := pyjson.Str(v)
			n, _ := counts.Get(tag)
			prev, _ := n.(int)
			counts.Set(tag, prev+1)
		}
	}
	return obj("source_state", source, "ack_state", ack, "records_counted", len(rows), "complete", !capped), "", elapsed
}

// tagCounts is each tag at zero, in order.
func tagCounts(tags []string) *pyjson.Object {
	o := pyjson.NewObject()
	for _, tag := range tags {
		o.Set(tag, 0)
	}
	return o
}
