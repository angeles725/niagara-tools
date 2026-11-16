package spy

import (
	"errors"
	"strings"
	"time"
)

// Table is one table of a page: the title (a lone heading cell above it), the header row and
// the rows of cells, as text.
type Table struct {
	Title  string
	Header []string
	Rows   [][]string
}

// Link is a link to another spy page: its text and its clean path below the spy root.
type Link struct{ Name, Path string }

// Page is a spy page as data. Actions are the POST forms it offers, by path, named only.
// Cut names the caps that dropped something; LinksSkipped counts links that leave the spy root.
type Page struct {
	Title        string
	NotFound     bool
	Tables       []Table
	Links        []Link
	Actions      []string
	Text         string
	LinksSkipped int
	Cut          []string
}

type parser struct {
	page    Page
	maxRows int
	rows    int
	stack   []*builder
	crumbs  []string
	link    *strings.Builder
	href    string
	heading *strings.Builder
	fields  int  // open form fields (fieldTags): their text is never kept
	option  bool // an option is open
	text    strings.Builder
	seen    map[string]bool // link paths kept
	acted   map[string]bool // action paths kept
}

// Limits bound one parse: MaxRows table rows are kept, all tables together (0: DefaultMaxRows),
// and the parse gives up with ErrDeadline once Deadline has passed (zero: no deadline).
type Limits struct {
	MaxRows  int
	Deadline time.Time
}

// ErrDeadline is a parse that did not finish by its deadline.
var ErrDeadline = errors.New("the spy page was not parsed before the deadline")

// Parse reads the HTML of a spy page within lim. A page that is not a spy page comes back with
// no tables and its text; the only failure is ErrDeadline.
func Parse(body string, lim Limits) (*Page, error) {
	maxRows := lim.MaxRows
	if maxRows <= 0 {
		maxRows = DefaultMaxRows
	}
	p := &parser{maxRows: maxRows, seen: map[string]bool{}, acted: map[string]bool{}}
	sc := &scanner{page: body}
	for n := 0; ; n++ {
		if n%deadlineEvery == 0 && !lim.Deadline.IsZero() && time.Now().After(lim.Deadline) {
			return nil, ErrDeadline
		}
		t, ok := sc.next()
		if !ok {
			break
		}
		switch {
		case t.name == "":
			p.addText(t.text)
		case t.end:
			p.closeTag(t.name)
		default:
			p.openTag(t)
		}
	}
	for len(p.stack) > 0 {
		p.closeTable()
	}
	p.page.Title = "Index"
	if n := len(p.crumbs); n > 0 {
		p.page.Title = p.crumbs[n-1]
	}
	text, cut := clip(lines(p.text.String()), MaxTextRunes)
	p.page.Text = text
	if cut {
		p.cut(CutText)
	}
	return &p.page, nil
}

func (p *parser) top() *builder {
	if len(p.stack) == 0 {
		return nil
	}
	return p.stack[len(p.stack)-1]
}

func (p *parser) cut(name string) {
	for _, c := range p.page.Cut {
		if c == name {
			return
		}
	}
	p.page.Cut = append(p.page.Cut, name)
}

// addText keeps s where it belongs: the open link and heading, then the open cell or, outside
// tables and links, the page text. Text of a form field is dropped wherever the field sits.
func (p *parser) addText(s string) {
	if p.fields > 0 || p.option {
		return
	}
	if p.link != nil {
		p.link.WriteString(s)
	}
	if p.heading != nil {
		p.heading.WriteString(s)
	}
	switch b := p.top(); {
	case b != nil && b.cell != nil:
		b.cell.WriteString(s)
	case b == nil && p.link == nil:
		p.text.WriteString(s)
	}
}

func (p *parser) openTag(t token) {
	if t.name == optionTag {
		p.option = true
		return
	}
	if fieldTags[t.name] {
		p.fields++
		return
	}
	switch t.name {
	case "table":
		p.stack = append(p.stack, &builder{nav: strings.Contains(t.attrs["class"], "spy-nav")})
	case "tr":
		if b := p.top(); b != nil {
			p.closeRow(b)
			b.inRow, b.header = true, true
		}
	case "td", "th":
		if b := p.top(); b != nil {
			p.closeCell(b)
			if !b.inRow {
				b.inRow, b.header = true, true
			}
			b.header = b.header && t.name == "th"
			if len(b.row) >= MaxRowCells {
				if !b.nav {
					p.cut(CutCells)
				}
				return
			}
			b.cell = &strings.Builder{}
		}
	case "a":
		if href, ok := t.attrs["href"]; ok {
			p.link, p.href = &strings.Builder{}, href
		}
	case "form":
		if strings.EqualFold(t.attrs["method"], "post") {
			p.addAction(t.attrs["action"])
		}
	case "h1", "h2", "h3":
		p.heading = &strings.Builder{}
	case "br", "p", "li", "hr", "div", "pre":
		p.addText("\n")
	}
}

func (p *parser) closeTag(name string) {
	if name == optionTag || name == selectTag {
		p.option = false
	}
	if fieldTags[name] {
		if p.fields > 0 {
			p.fields--
		}
		return
	}
	switch name {
	case "table":
		if len(p.stack) > 0 {
			p.closeTable()
		}
	case "tr":
		if b := p.top(); b != nil {
			p.closeRow(b)
		}
	case "td", "th":
		if b := p.top(); b != nil {
			p.closeCell(b)
		}
	case "a":
		p.closeLink()
	case "h1", "h2", "h3":
		if p.heading != nil && oneLine(p.heading.String()) == notFoundHeading {
			p.page.NotFound = true
		}
		p.heading = nil
		p.addText("\n")
	case "p", "li", "div", "pre", "ul":
		p.addText("\n")
	}
}
