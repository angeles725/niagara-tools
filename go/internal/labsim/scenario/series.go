package scenario

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// MaxReplayRecords bounds the records a replay file may hold.
const MaxReplayRecords = 1_000_000

// Series is a recorded series of a replay file: the records whose status is ok,
// oldest first.
type Series struct {
	Times   []time.Time
	Values  []float64
	Skipped int // records left out for a status other than ok
}

// columns are the places of the time, value and status columns (status -1 when
// the file has none).
type columns struct{ time, value, status int }

// ReadSeries reads a replay file: a CSV with a header naming a time column
// (time or timestamp, so the Timestamp of a history read as ITableToCsv), a
// value column and, optionally, a status column. A time carries its zone
// (2026-11-15T20:00:00.000-06:00 or ...Z), a value is a plain number. A record
// whose status is neither ok, {ok} nor empty is skipped; the rest are sorted
// oldest first. The error is one line naming the row and column.
func ReadSeries(r io.Reader) (*Series, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // a row of another width is named below, with both counts
	header, err := cr.Read()
	if errors.Is(err, io.EOF) {
		return nil, errors.New(msgSeriesNoHeader)
	}
	if err != nil {
		return nil, csvFault(err)
	}
	cols, err := findColumns(header)
	if err != nil {
		return nil, err
	}
	s := &Series{}
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, csvFault(err)
		}
		row, _ := cr.FieldPos(0)
		if len(rec) != len(header) {
			return nil, fmt.Errorf(msgSeriesFields, row, len(rec), len(header))
		}
		if len(s.Values)+s.Skipped >= MaxReplayRecords {
			return nil, fmt.Errorf(msgSeriesTooLong, row, MaxReplayRecords)
		}
		if cols.status >= 0 && !statusOK(rec[cols.status]) {
			s.Skipped++
			continue
		}
		at, value, err := record(rec, header, cols, row)
		if err != nil {
			return nil, err
		}
		s.Times, s.Values = append(s.Times, at), append(s.Values, value)
	}
	if len(s.Values) == 0 {
		return nil, fmt.Errorf(msgSeriesNoOK, s.Skipped)
	}
	s.sort()
	return s, nil
}

// findColumns places the columns by their header names, ignoring case, a
// byte-order mark and anything that is not a letter.
func findColumns(header []string) (columns, error) {
	c := columns{time: -1, value: -1, status: -1}
	for i, name := range header {
		key := strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, name)
		switch {
		case c.time < 0 && (key == "time" || strings.HasPrefix(key, "timestamp")):
			c.time = i
		case c.value < 0 && strings.HasPrefix(key, "value"):
			c.value = i
		case c.status < 0 && key == "status":
			c.status = i
		}
	}
	switch {
	case c.time < 0:
		return c, errors.New(msgSeriesNoTime)
	case c.value < 0:
		return c, errors.New(msgSeriesNoValue)
	}
	return c, nil
}

func statusOK(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return s == "" || s == "ok" || s == "{ok}"
}

// record is the time and value of one row.
func record(rec, header []string, c columns, row int) (time.Time, float64, error) {
	text := strings.TrimSpace(rec[c.time])
	at, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return at, 0, fmt.Errorf(msgSeriesTime, row, c.time+1, strings.TrimSpace(header[c.time]), text)
	}
	text = strings.TrimSpace(rec[c.value])
	value, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return at, 0, fmt.Errorf(msgSeriesValue, row, c.value+1, strings.TrimSpace(header[c.value]), text)
	}
	return at, value, nil
}

// csvFault is a CSV syntax error as one line naming the row and column.
func csvFault(err error) error {
	var pe *csv.ParseError
	if !errors.As(err, &pe) {
		return err
	}
	return fmt.Errorf(msgSeriesSyntax, pe.StartLine, pe.Line, pe.Column, pe.Err)
}

// sort orders the records oldest first, keeping file order between equal times.
func (s *Series) sort() {
	idx := make([]int, len(s.Times))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return s.Times[idx[a]].Before(s.Times[idx[b]]) })
	times, values := make([]time.Time, len(idx)), make([]float64, len(idx))
	for i, j := range idx {
		times[i], values[i] = s.Times[j], s.Values[j]
	}
	s.Times, s.Values = times, values
}
