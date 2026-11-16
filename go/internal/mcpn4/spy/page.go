package spy

import (
	"strings"
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
	forms   int
	text    strings.Builder
	seen    map[string]bool
}

// Parse reads the HTML of a spy page, keeping at most maxRows table rows (0: DefaultMaxRows).
// It never fails: a page that is not a spy page comes back with no tables and its text.
func Parse(body string, maxRows int) *Page {
	if maxRows <= 0 {
		maxRows = DefaultMaxRows
	}
	p := &parser{maxRows: maxRows, seen: map[string]bool{}}
	for _, t := range tokens(body) {
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
	return &p.page
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

func (p *parser) addText(s string) {
	if p.link != nil {
		p.link.WriteString(s)
	}
	if p.heading != nil {
		p.heading.WriteString(s)
	}
	switch b := p.top(); {
	case b != nil && b.cell != nil:
		b.cell.WriteString(s)
	case b == nil && p.forms == 0 && p.link == nil:
		p.text.WriteString(s)
	}
}

func (p *parser) openTag(t token) {
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
			b.cell = &strings.Builder{}
			b.header = b.header && t.name == "th"
		}
	case "a":
		if href, ok := t.attrs["href"]; ok {
			p.link, p.href = &strings.Builder{}, href
		}
	case "form":
		p.forms++
		if strings.EqualFold(t.attrs["method"], "post") {
			if action, ok := underRoot(t.attrs["action"]); ok && action != "" {
				p.page.Actions = append(p.page.Actions, oneLine(action))
			}
		}
	case "h1", "h2", "h3":
		p.heading = &strings.Builder{}
	case "br", "p", "li", "hr", "div", "pre":
		p.addText("\n")
	}
}

func (p *parser) closeTag(name string) {
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
	case "form":
		if p.forms > 0 {
			p.forms--
		}
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

// closeLink files the open link: a breadcrumb inside the navigation table, else a child link.
func (p *parser) closeLink() {
	if p.link == nil {
		return
	}
	name, href := oneLine(p.link.String()), p.href
	p.link = nil
	if b := p.top(); b != nil && b.nav {
		p.crumbs = append(p.crumbs, name)
		return
	}
	path, ok := linkPath(href)
	switch {
	case !ok:
		p.page.LinksSkipped++
	case p.seen[path]:
	case len(p.page.Links) >= MaxLinks:
		p.cut(CutLinks)
	default:
		p.seen[path] = true
		p.page.Links = append(p.page.Links, Link{Name: name, Path: path})
	}
}
