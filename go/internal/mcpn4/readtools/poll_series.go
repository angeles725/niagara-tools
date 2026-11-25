package readtools

import (
	"strconv"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/box"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// pollSeriesMax caps the summary entries (rows x columns) of one poll.
const pollSeriesMax = 200

// pollSeries follows each (row, column) cell across the samples: a row is named by its first
// column (a slot path in a select slotPath, ...), every other column is a cell.
type pollSeries struct {
	order []string
	cells map[string]*pollCell
}

type pollCell struct {
	row, column, first, last string
	changes, seen            int
	min, max                 float64
	numeric                  bool
}

func newPollSeries() *pollSeries { return &pollSeries{cells: map[string]*pollCell{}} }

func (p *pollSeries) add(columns []string, rows []*pyjson.Object) {
	if len(columns) < 2 {
		return
	}
	for _, r := range rows {
		key, _ := r.Get(columns[0])
		for _, col := range columns[1:] {
			v, _ := r.Get(col)
			p.note(pyjson.Str(key), col, pyjson.Str(v))
		}
	}
}

func (p *pollSeries) note(row, column, value string) {
	id := row + "\x00" + column
	c, ok := p.cells[id]
	if !ok {
		if len(p.order) >= pollSeriesMax {
			return
		}
		c = &pollCell{row: row, column: column, first: value, last: value, numeric: true}
		p.cells[id], p.order = c, append(p.order, id)
	} else if value != c.last {
		c.changes++
		c.last = value
	}
	c.seen++
	f, err := strconv.ParseFloat(strings.Replace(strings.Fields(value + " x")[0], ",", ".", 1), 64)
	if err != nil {
		c.numeric = false
		return
	}
	if c.seen == 1 || f < c.min {
		c.min = f
	}
	if c.seen == 1 || f > c.max {
		c.max = f
	}
}

// summary is one entry per cell: first and last value, how many times it changed, and min and
// max when every value read as a number (a locale decimal comma and a trailing unit allowed).
func (p *pollSeries) summary() []any {
	out := []any{}
	for _, id := range p.order {
		c := p.cells[id]
		e := obj("row", c.row, "column", c.column, "first", c.first, "last", c.last, "changes", c.changes)
		if c.numeric {
			e.Set("min", c.min)
			e.Set("max", c.max)
		}
		out = append(out, e)
	}
	return out
}

// stationClock reads the SystemService systemTime once: the station's time, this machine's time
// at the middle of the read, offset_s (station minus local) and uncertainty_s (half the round
// trip). A read that fails or a time that does not parse is a note, never a guessed offset.
func (t *Tools) stationClock() *pyjson.Object {
	before := t.now()
	nodes, _, err := t.timedLoad(systemServiceOrd, 2)
	after := t.now()
	if err != nil {
		return obj("note", "the station time could not be read: "+err.Error())
	}
	slots := pyjson.NewObject()
	if systemSlotsFrom(nodes, slots, []string{"systemTime"}) != nil {
		return obj("note", "the station time could not be read")
	}
	text, _ := slots.Get("systemTime")
	at, perr := time.Parse(time.RFC3339Nano, pyjson.Str(text))
	if box.Node(box.Children(nodes, ""), "systemTime") == nil || perr != nil {
		return obj("note", "the SystemService answered no whole systemTime", "station_time", text)
	}
	mid := before.Add(after.Sub(before) / 2)
	return obj("station_time", pyjson.Str(text), "local_time", mid.Format(time.RFC3339Nano),
		"offset_s", round3(at.Sub(mid).Seconds()), "uncertainty_s", round3(after.Sub(before).Seconds()/2))
}
