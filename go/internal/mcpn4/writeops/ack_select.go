package writeops

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// n4_ack_alarm by selection (station-audits S6): select {source?, alarm_class?, before?, max?}
// is resolved in the dry run to the exact uuids it matches, with n4_read_alarms' own records
// query and where clause (bql.AlarmWhere: every operator value checked on its BQL-decoded form),
// unacknowledged records only, newest first.

// Messages of the selection, as data.
const (
	msgSelBoth     = "give uuids or select, not both"
	msgSelNone     = "give uuids or select: one of them is required"
	msgSelObject   = "select must be an object {source?, alarm_class?, before?, max?}"
	msgSelEmpty    = "select needs at least one of source, alarm_class or before"
	msgSelUnknown  = "select.%s is not a selection key (source, alarm_class, before, max)"
	msgSelString   = "select.%s must be a string"
	msgSelSource   = "select.source must be a slot ORD such as station:|slot:/Drivers/Net (slot:/a and local:|station:|slot:/a too), got %q"
	msgSelBefore   = "select.before must be a BAbsTime such as 2026-11-15T00:00:00.000-06:00 or a span back from now such as 24h, got %q"
	msgSelMax      = "select.max must be a whole number from 1 to %d"
	msgSelBadUUID  = "the station returned a record uuid that is not a uuid: %q"
	noteSelNothing = "no unacknowledged alarm record matches the selection: nothing to acknowledge, no token issued"
)

// selectKeys are the keys a selection may hold, in the order its criteria echo them.
var selectKeys = []string{"source", "alarm_class", "before", "max"}

// ackSelection is a parsed select: the where clause it reads with, the echo of its criteria
// and the most records it acknowledges.
type ackSelection struct {
	where    string
	criteria *pyjson.Object
	max      int
}

// parseSelect checks v before anything reaches the station.
func parseSelect(v any) (*ackSelection, error) {
	o, ok := v.(*pyjson.Object)
	if !ok {
		return nil, plan.ToolError(msgSelObject)
	}
	s, f := &ackSelection{criteria: pyjson.NewObject(), max: maxAckIDs}, bql.AlarmFilter{AckState: "unacked"}
	for _, k := range o.Keys() {
		if !containsName(selectKeys, k) {
			return nil, plan.ToolError(fmt.Sprintf(msgSelUnknown, k))
		}
	}
	for _, k := range selectKeys[:3] {
		raw, has := o.Get(k)
		if !has {
			continue
		}
		text, isStr := raw.(string)
		if !isStr {
			return nil, plan.ToolError(fmt.Sprintf(msgSelString, k))
		}
		switch k {
		case "source":
			path, ok := bql.RecordSourcePath(text)
			if !ok {
				return nil, plan.ToolError(fmt.Sprintf(msgSelSource, text))
			}
			f.Source, text = "slot:"+path, "station:|slot:"+path
		case "alarm_class":
			f.AlarmClass = text
		case "before":
			if _, _, ok := bql.ResolveTime(text); !ok {
				return nil, plan.ToolError(fmt.Sprintf(msgSelBefore, text))
			}
			f.Until = text
		}
		s.criteria.Set(k, text)
	}
	if len(s.criteria.Keys()) == 0 {
		return nil, plan.ToolError(msgSelEmpty)
	}
	if raw, has := o.Get("max"); has {
		n, isNum := raw.(pyjson.Number)
		big, isInt := n.BigInt()
		if !isNum || !isInt || !big.IsInt64() || big.Int64() < 1 || big.Int64() > maxAckIDs {
			return nil, plan.ToolError(fmt.Sprintf(msgSelMax, maxAckIDs))
		}
		s.max = int(big.Int64())
	}
	where, err := bql.AlarmWhere(f)
	if err != nil {
		return nil, plan.ToolError(err.Error())
	}
	s.where = where
	return s, nil
}

// resolve is the selection's uuids and its report {criteria, where, records, matched,
// truncated, max}: records newest first, each {uuid, source, alarm_class, timestamp}.
func (s *ackSelection) resolve(r ordReader) ([]any, *pyjson.Object, error) {
	text, cut, err := r.GetOrd(bql.ComposeAlarmOrd(bql.AlarmRecordsQuery, s.where, bql.AlarmOrder), ackTimeout,
		bql.MaxResponseBytes)
	if err != nil {
		return nil, nil, err
	}
	if cut {
		text = bql.CompleteRecords(text)
	}
	columns, rows, capped, err := bql.ParseCSV(text, s.max, []int{})
	if err != nil {
		return nil, nil, err
	}
	keyed, err := bql.ByPosition(columns, rows, bql.AlarmRecordFields)
	if err != nil {
		return nil, nil, err
	}
	ids, records := []any{}, []any{}
	for _, row := range keyed {
		id := strings.ToLower(pyjson.Str(get(row, "uuid")))
		if !uuidRe.MatchString(id) {
			return nil, nil, plan.ToolError(fmt.Sprintf(msgSelBadUUID, id))
		}
		ids = append(ids, id)
		records = append(records, obj("uuid", id, "source", get(row, "source"), "alarm_class", get(row, "alarm_class"),
			"timestamp", bql.IsoOrNull(pyjson.Str(get(row, "timestamp_iso")))))
	}
	matched, err := s.count(r)
	if err != nil {
		return nil, nil, err
	}
	truncated := cut || capped || matched != nil && matched.(int) > len(ids)
	return ids, obj("criteria", s.criteria, "where", s.where, "records", records, "matched", matched,
		"truncated", truncated, "max", s.max), nil
}

// count is the number of records the selection matches (select count(*)), nil when the station
// answered no whole number.
func (s *ackSelection) count(r ordReader) (any, error) {
	text, _, err := r.GetOrd(bql.ComposeAlarmOrd(bql.AlarmCountQuery, s.where, ""), ackTimeout, bql.MaxResponseBytes)
	if err != nil {
		return nil, err
	}
	_, rows, _, err := bql.ParseCSV(text, 1, []int{})
	if err != nil || len(rows) != 1 || len(rows[0].Keys()) != 1 {
		return nil, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(pyjson.Str(get(rows[0], rows[0].Keys()[0]))))
	if err != nil || n < 0 {
		return nil, nil
	}
	return n, nil
}

// nothingSelected is the dry run reply of a selection that matched no record: no plan, no token.
func nothingSelected(report *pyjson.Object) *pyjson.Object {
	return obj("dry_run", true, "selection", report, "note", noteSelNothing)
}
