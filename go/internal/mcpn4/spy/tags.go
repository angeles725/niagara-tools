package spy

import "strings"

// What a start tag and an end tag do to the page being read: tables, rows and cells, links,
// form fields and actions, headings and line breaks.

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
