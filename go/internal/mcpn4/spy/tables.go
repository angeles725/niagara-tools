package spy

import "strings"

// builder is a table being read.
type builder struct {
	nav    bool
	table  Table
	row    []string
	header bool // every cell of the open row is a th
	inRow  bool
	cell   *strings.Builder
}

func (p *parser) closeCell(b *builder) {
	if b.cell == nil {
		return
	}
	text := ""
	if !b.nav { // a navigation table is read for its links only
		var cut bool
		if text, cut = clip(oneLine(b.cell.String()), MaxCellRunes); cut {
			p.cut(CutCell)
		}
	}
	b.row, b.cell = append(b.row, text), nil
}

// closeRow files the open row: a lone heading cell before any row is the table's title, a row
// of heading cells before any row its header, anything else a row (up to the cap).
func (p *parser) closeRow(b *builder) {
	p.closeCell(b)
	row, header := b.row, b.header
	b.row, b.inRow = nil, false
	t := &b.table
	switch {
	case len(row) == 0 || b.nav:
	case header && len(row) == 1 && t.Title == "" && t.Header == nil && len(t.Rows) == 0:
		t.Title = row[0]
	case header && t.Header == nil && len(t.Rows) == 0:
		t.Header = row
	case p.rows >= p.maxRows:
		p.cut(CutRows)
	default:
		p.rows++
		t.Rows = append(t.Rows, row)
	}
}

func (p *parser) closeTable() {
	b := p.top()
	p.closeRow(b)
	p.stack = p.stack[:len(p.stack)-1]
	switch {
	case b.nav:
	case len(p.page.Tables) >= MaxTables:
		p.cut(CutTables)
	default:
		p.page.Tables = append(p.page.Tables, b.table)
	}
}
