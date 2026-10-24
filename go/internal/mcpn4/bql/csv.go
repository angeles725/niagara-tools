package bql

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// FieldLimit is csv.field_size_limit(): a longer field is csv.Error, not a cut cell.
const FieldLimit = 131072

// csvError is the csv.Error the reader raises: the server shows only its type.
var csvError = &pyjson.Exception{Type: "Error"}

// The states of _csv.c's reader with the excel dialect (no escapechar, QUOTE_MINIMAL, not
// strict); eol is the end-of-line sentinel it processes after every line.
const (
	startRecord = iota
	startField
	inField
	inQuotedField
	quoteInQuotedField
	eatCRNL
	eol = rune(-2)
)

// reader is csv.reader(io.StringIO(text, newline="")): lines end at \n, \r or \r\n, and a
// quoted cell may span lines.
type reader struct {
	text   string
	pos    int
	state  int
	fields []string
	field  strings.Builder
	n      int // code points in field
}

// line is the next line of the StringIO, its terminator included; ok is false at the end.
func (r *reader) line() (string, bool) {
	if r.pos >= len(r.text) {
		return "", false
	}
	rest := r.text[r.pos:]
	end := strings.IndexAny(rest, "\r\n")
	switch {
	case end < 0:
		end = len(rest)
	case rest[end] == '\r' && end+1 < len(rest) && rest[end+1] == '\n':
		end += 2
	default:
		end++
	}
	r.pos += end
	return rest[:end], true
}

func (r *reader) save() {
	r.fields = append(r.fields, r.field.String())
	r.field.Reset()
	r.n = 0
}

func (r *reader) add(c rune) error {
	if r.n >= FieldLimit {
		return csvError
	}
	r.field.WriteString(pyjson.Chr(c))
	r.n++
	return nil
}

// endOfLine saves the field at a line end and moves to the next record (eol) or eats \r\n.
func (r *reader) endOfLine(c rune) {
	r.save()
	r.state = eatCRNL
	if c == eol {
		r.state = startRecord
	}
}

// process is parse_process_char.
func (r *reader) process(c rune) error {
	nl := c == '\n' || c == '\r'
	switch r.state {
	case startRecord:
		if c == eol {
			return nil
		}
		if nl {
			r.state = eatCRNL
			return nil
		}
		r.state = startField
		fallthrough
	case startField:
		switch {
		case nl || c == eol:
			r.endOfLine(c)
		case c == '"':
			r.state = inQuotedField
		case c == ',':
			r.save()
		default:
			r.state = inField
			return r.add(c)
		}
	case inField:
		switch {
		case nl || c == eol:
			r.endOfLine(c)
		case c == ',':
			r.save()
			r.state = startField
		default:
			return r.add(c)
		}
	case inQuotedField:
		switch {
		case c == eol:
		case c == '"':
			r.state = quoteInQuotedField
		default:
			return r.add(c)
		}
	case quoteInQuotedField:
		switch {
		case c == '"':
			r.state = inQuotedField
			return r.add(c)
		case c == ',':
			r.save()
			r.state = startField
		case nl || c == eol:
			r.endOfLine(c)
		default:
			r.state = inField
			return r.add(c)
		}
	case eatCRNL:
		if c == eol {
			r.state = startRecord
		}
	}
	return nil
}

// next is the reader's __next__: the next record ([] for an empty line), ok false at the end.
func (r *reader) next() ([]string, bool, error) {
	r.fields, r.state, r.n = []string{}, startRecord, 0
	r.field.Reset()
	for {
		line, ok := r.line()
		if !ok {
			if r.n != 0 || r.state == inQuotedField {
				r.save()
				return r.fields, true, nil
			}
			return nil, false, nil
		}
		for i := 0; i < len(line); {
			c, size := pyjson.DecodeRune(line[i:])
			i += size
			if err := r.process(c); err != nil {
				return nil, false, err
			}
		}
		if err := r.process(eol); err != nil {
			return nil, false, err
		}
		if r.state == startRecord {
			return r.fields, true, nil
		}
	}
}

// ParseCSV is parse_csv: the columns, rows (by column) and whether rows were capped at maxRows,
// from an ITableToCsv body. Repeated headers are made unique; the slot-name cells decoded are
// those at namePositions or, when it is nil, under a NameColumns header. A body without a
// header row and a record whose cell count differs from the header's are a ValueError.
func ParseCSV(text string, maxRows int, namePositions []int) ([]string, []*pyjson.Object, bool, error) {
	r := &reader{text: strings.TrimLeft(text, "\ufeff")}
	header, _, err := r.next()
	if err != nil {
		return nil, nil, false, err
	}
	if len(header) == 0 {
		return nil, nil, false, valueError("the BQL answer has no header row (empty body): it is not a CSV " +
			"table, so no rows can be read from it")
	}
	names := make([]string, len(header))
	for i, h := range header {
		names[i] = Strip(StripControl(h))
	}
	columns := UniqueColumns(names)
	decode := map[int]bool{}
	for i, col := range columns {
		if namePositions == nil && contains(NameColumns, col) {
			decode[i] = true
		}
	}
	for _, i := range namePositions {
		decode[i] = true
	}
	rows := []*pyjson.Object{}
	for {
		record, ok, err := r.next()
		if err != nil {
			return nil, nil, false, err
		}
		if !ok {
			return columns, rows, false, nil
		}
		if len(record) == 0 {
			continue
		}
		if len(rows) >= maxRows {
			return columns, rows, true, nil
		}
		if len(record) != len(columns) {
			return nil, nil, false, valueError(fmt.Sprintf("the BQL answer is ragged: record %d has %d cell(s), "+
				"the header has %d", len(rows)+1, len(record), len(columns)))
		}
		row := pyjson.NewObject()
		for i, col := range columns {
			cell := StripControl(record[i])
			if decode[i] {
				cell = UnescapeSlot(cell)
			}
			row.Set(col, cell)
		}
		rows = append(rows, row)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
