package readtools

import (
	"sort"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The grouping of n4_alarm_health, pure over the keyed rows of its two reads. The alarm
// database keeps one record per alarm event (bql/event.go): a record whose alarm time is in the
// window is one transition out of normal, and its normal time, when set, one transition back.

// alarmSource is the source a record names, one spelling per component: a slot path stored as
// slot:/a, station:|slot:/a or local:|station:|slot:/a (bql/alarm.go) reads station:|slot:/a, the
// form the source filter of n4_read_alarms and n4_event_report takes; any other text is kept.
func alarmSource(stored string) string {
	if path, ok := bql.RecordSourcePath(stored); ok {
		return "station:|slot:" + path
	}
	return stored
}

// sourceTally is one source's records in the window and the transitions they hold.
type sourceTally struct {
	source               string
	records, transitions int
}

// tallySources counts the window rows ({source, normal_time}) per source, in first-seen order.
func tallySources(rows []*pyjson.Object) []*sourceTally {
	var out []*sourceTally
	bySource := map[string]*sourceTally{}
	for _, row := range rows {
		src := alarmSource(cell(row, "source"))
		s, ok := bySource[src]
		if !ok {
			s = &sourceTally{source: src}
			bySource[src] = s
			out = append(out, s)
		}
		s.records++
		s.transitions++
		if bql.IsoOrNull(cell(row, "normal_time")) != nil {
			s.transitions++
		}
	}
	return out
}

// chatteringSources are the sources with at least threshold transitions, most first (then by
// source), as {source, transitions, records}.
func chatteringSources(tallies []*sourceTally, threshold int) []any {
	var hit []*sourceTally
	for _, s := range tallies {
		if s.transitions >= threshold {
			hit = append(hit, s)
		}
	}
	sort.SliceStable(hit, func(i, j int) bool {
		if hit[i].transitions != hit[j].transitions {
			return hit[i].transitions > hit[j].transitions
		}
		return hit[i].source < hit[j].source
	})
	out := []any{}
	for _, s := range hit {
		out = append(out, obj("source", s.source, "transitions", s.transitions, "records", s.records))
	}
	return out
}

// busiestSources are every source of the window by record count, most first (then by source),
// as {source, count}.
func busiestSources(tallies []*sourceTally) []any {
	sorted := append([]*sourceTally(nil), tallies...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].records != sorted[j].records {
			return sorted[i].records > sorted[j].records
		}
		return sorted[i].source < sorted[j].source
	})
	out := []any{}
	for _, s := range sorted {
		out = append(out, obj("source", s.source, "count", s.records))
	}
	return out
}

// openAlarm is one open record with its alarm time read.
type openAlarm struct {
	row *pyjson.Object
	at  time.Time
	ok  bool
}

// openFindings splits the open rows ({source, source_state, ack_state, alarm_class,
// alarm_time}): offnormal are the sources with a record not back to normal, once per source
// ({source, source_state, alarm_class, since, records}, state and class of its earliest record),
// unacked the records not acknowledged whose alarm time is more than days days before now
// ({source, alarm_class, ack_state, alarm_time, age_days}), both oldest first; unreadable counts
// the records whose alarm time could not be read (left out of unacked).
func openFindings(rows []*pyjson.Object, now time.Time, days int) (offnormal, unacked []any, unreadable int) {
	var all []openAlarm
	for _, row := range rows {
		at, err := time.Parse(isoLayout, cell(row, "alarm_time"))
		all = append(all, openAlarm{row: row, at: at, ok: err == nil})
		if err != nil {
			unreadable++
		}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].ok && (!all[j].ok || all[i].at.Before(all[j].at)) })
	offnormal, unacked = []any{}, []any{}
	bySource := map[string]*pyjson.Object{}
	limit := time.Duration(days) * 24 * time.Hour
	for _, a := range all {
		src := alarmSource(cell(a.row, "source"))
		if cell(a.row, "source_state") != "normal" {
			if seen, ok := bySource[src]; ok {
				seen.Set("records", get(seen, "records").(int)+1)
			} else {
				bySource[src] = obj("source", src, "source_state", cell(a.row, "source_state"),
					"alarm_class", cell(a.row, "alarm_class"), "since", bql.IsoOrNull(cell(a.row, "alarm_time")), "records", 1)
				offnormal = append(offnormal, bySource[src])
			}
		}
		if cell(a.row, "ack_state") == "acked" || !a.ok || now.Sub(a.at) <= limit {
			continue
		}
		unacked = append(unacked, obj("source", src, "alarm_class", cell(a.row, "alarm_class"),
			"ack_state", cell(a.row, "ack_state"), "alarm_time", cell(a.row, "alarm_time"),
			"age_days", int(now.Sub(a.at)/(24*time.Hour))))
	}
	return offnormal, unacked, unreadable
}
