package readtools

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/bql"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The history side of n4_event_report: each event's window, the history that answers for its
// source (history_id, or the history extension found beside the alarm source once per source)
// and the n4_read_history summary over that window. A missing history or a failed read is a null
// summary with a note on that event; it never fails the report.

// Layouts of the window: shown in the alarm time's offset, sent to the history as UTC.
const (
	windowShown = "2006-01-02T15:04:05.000Z07:00"
	windowSent  = "2006-01-02T15:04:05.000Z"
	isoLayout   = "2006-01-02T15:04:05.999999999Z07:00"
)

// Messages of the history side, as data.
const (
	msgEventNoSlot  = "the alarm source %s is not a slot path: give history_id"
	msgEventNoExt   = "no history extension on the alarm source or beside it (%s at %s): give history_id"
	msgEventExtFail = "the history extension lookup (%s at %s) failed: %s"
	msgEventNoTime  = "the record has no alarm time"
)

// eventWindows reads the history of each event, finding a source's history once per report.
type eventWindows struct {
	t               *Tools
	historyID       string
	margin, timeout int
	found           map[string]*historyFound
	elapsed         int
}

// historyFound is the history that answers for a source, or a note saying why there is none.
type historyFound struct {
	id, note string
	shown    *pyjson.Object
}

// report is one event of the reply: its facts, window, history and summary.
func (w *eventWindows) report(ev *event) *pyjson.Object {
	o := eventFacts(ev)
	start, end, isNow, err := w.window(cell(ev.row, "alarm_time"), cell(ev.row, "normal_time"))
	if err != nil {
		o.Set("window", nil)
		o.Set("history", nil)
		o.Set("summary", nil)
		o.Set("summary_note", windowNote(err))
		return o
	}
	o.Set("window", obj("start", start.Format(windowShown), "end", end.Format(windowShown), "end_is_now", isNow))
	h := w.history(cell(ev.row, "source"))
	o.Set("history", nilIfNone(h.shown))
	if h.id == "" {
		o.Set("summary", nil)
		o.Set("summary_note", h.note)
		return o
	}
	params, err := bql.HistoryParams("", start.UTC().Format(windowSent), end.UTC().Format(windowSent))
	var summary any
	note, ms := "", 0
	if err != nil {
		note = err.Error()
	} else {
		summary, note, ms = w.t.historySummary(h.id, params, eventSummaryRows, w.timeout, nil)
	}
	w.elapsed += ms
	o.Set("summary", summary)
	if note != "" {
		o.Set("summary_note", note)
	}
	return o
}

// window is [alarm time - margin, normal time + margin], or up to now while the source has not
// returned to normal, in the alarm time's offset.
func (w *eventWindows) window(alarmText, normalText string) (time.Time, time.Time, bool, error) {
	if bql.IsoOrNull(alarmText) == nil {
		return time.Time{}, time.Time{}, false, fmt.Errorf("%s", msgEventNoTime)
	}
	at, err := time.Parse(isoLayout, alarmText)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	margin := time.Duration(w.margin) * time.Minute
	if bql.IsoOrNull(normalText) == nil {
		return at.Add(-margin), bql.Now().In(at.Location()), true, nil
	}
	normal, err := time.Parse(isoLayout, normalText)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}
	return at.Add(-margin), normal.Add(margin).In(at.Location()), false, nil
}

// history is the history of source: history_id when given, else the history extension on the
// alarm source's component (the alarm extension's parent) or on the source itself.
func (w *eventWindows) history(source string) *historyFound {
	if w.historyID != "" {
		return &historyFound{id: w.historyID, shown: obj("id", w.historyID, "found_by", "history_id")}
	}
	slot, ok := bql.RecordSourcePath(source)
	if !ok {
		return &historyFound{note: fmt.Sprintf(msgEventNoSlot, pyjson.StrRepr(source))}
	}
	if h, ok := w.found[slot]; ok {
		return h
	}
	h := w.findExt(slot)
	w.found[slot] = h
	return h
}

// findExt is the first history extension whose parent is slot's parent or slot itself, from one
// lookup at slot's parent; the other matches are listed as other_exts.
func (w *eventWindows) findExt(slot string) *historyFound {
	parent := path.Dir(slot)
	base := bql.StationSlot + parent
	columns, rows, _, ms, err := w.t.query(base, historyExtQuery, bql.DefaultMaxRows, w.timeout)
	w.elapsed += ms
	var keyed []*pyjson.Object
	if err == nil {
		keyed, err = bql.ByPosition(columns, rows, []string{"slot", "id"})
	}
	if err != nil {
		return &historyFound{note: fmt.Sprintf(msgEventExtFail, historyExtQuery, base, err.Error())}
	}
	var h *historyFound
	var others []any
	for _, row := range keyed {
		ext := cell(row, "slot")
		if dir := path.Dir(strings.TrimPrefix(ext, "slot:")); dir != parent && dir != slot {
			continue
		}
		if h != nil {
			others = append(others, ext)
			continue
		}
		h = &historyFound{id: cell(row, "id"), shown: obj("id", cell(row, "id"), "found_by", "history_ext", "ext_slot", ext)}
	}
	if h == nil {
		return &historyFound{note: fmt.Sprintf(msgEventNoExt, historyExtQuery, base)}
	}
	if len(others) > 0 {
		h.shown.Set("other_exts", others)
	}
	return h
}
