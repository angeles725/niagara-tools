// Start tags, end tags and namespaces of the expat-compatible parser (see pyxml.go).

package pyxml

import "strings"

// startTag parses "<name attr='v'...>" or "/>" from its '<'.
func (p *xparser) startTag(s int) (int, error) {
	if !qStart(p.at(s + 1)) { // the prolog hands over any non-ASCII character after '<'
		return s, p.fail(s+1, errInvalid)
	}
	i, err := p.scanQName(s + 1)
	if err != nil {
		return s, err
	}
	qname := string(p.r[s+1 : i])
	var raws []rawAttr
	empty := false
	for done := false; !done; {
		sawS := false
		for !p.eof(i) && isS(p.r[i]) {
			i, sawS = i+1, true
		}
		if p.eof(i) {
			return s, p.fail(s, errUnclosed)
		}
		switch c := p.r[i]; {
		case c == '>':
			i, done = i+1, true
		case c == '/':
			if p.eof(i + 1) {
				return s, p.fail(s, errUnclosed)
			}
			if p.r[i+1] != '>' {
				return s, p.fail(i+1, errInvalid)
			}
			i, done, empty = i+2, true, true
		case sawS && qStart(c):
			a, next, err := p.attribute(s, i)
			if err != nil {
				return s, err
			}
			raws, i = append(raws, a), next
		default:
			return s, p.fail(i, errInvalid)
		}
	}
	return i, p.openElement(s, qname, raws, empty)
}

// attribute scans name S? = S? quoted-value starting at the name.
func (p *xparser) attribute(s, i int) (rawAttr, int, error) {
	a := rawAttr{pos: i}
	i, err := p.scanQName(i)
	if err != nil {
		return a, i, err
	}
	a.name = string(p.r[a.pos:i])
	for !p.eof(i) && isS(p.r[i]) {
		i++
	}
	if p.eof(i) {
		return a, i, p.fail(s, errUnclosed)
	}
	if p.r[i] != '=' {
		return a, i, p.fail(i, errInvalid)
	}
	for i++; !p.eof(i) && isS(p.r[i]); i++ {
	}
	if p.eof(i) {
		return a, i, p.fail(s, errUnclosed)
	}
	q := p.r[i]
	if q != '"' && q != '\'' {
		return a, i, p.fail(i, errInvalid)
	}
	a.vs = i + 1
	for i = a.vs; ; {
		if p.eof(i) {
			return a, i, p.fail(s, errUnclosed)
		}
		switch c := p.r[i]; {
		case c == q:
			a.ve = i
			return a, i + 1, nil
		case c == '<' || c == bad || !isXMLChar(c):
			return a, i, p.fail(i, errInvalid)
		case c == '&':
			e, err := p.refSyntax(i, s)
			if err != nil {
				return a, i, err
			}
			i = e
		default:
			i++
		}
	}
}

// openElement checks the attributes of a complete start tag, builds the element and opens it.
func (p *xparser) openElement(s int, qname string, raws []rawAttr, empty bool) error {
	el := &Element{Tag: qname}
	seen := make(map[string]bool, len(raws))
	for _, a := range raws {
		if seen[a.name] {
			return p.fail(a.pos, errDupAttr)
		}
		seen[a.name] = true
		v, err := p.attrValue(s, a)
		if err != nil {
			return err
		}
		el.Attrs = append(el.Attrs, Attr{a.name, v})
	}
	var scope map[string]string
	if len(p.stack) > 0 {
		scope = p.stack[len(p.stack)-1].ns
	}
	scope, err := p.namespaces(s, el, scope)
	if err != nil {
		return err
	}
	if len(p.stack) == 0 {
		p.root = el
	} else {
		parent := p.stack[len(p.stack)-1].elem
		parent.Children = append(parent.Children, el)
	}
	if !empty {
		p.stack = append(p.stack, &openElem{qname: qname, elem: el, ns: scope})
	}
	return nil
}

// namespaces applies the element's xmlns declarations and expands its names (ElementTree's
// "{uri}local"); it returns the scope for the element's content.
func (p *xparser) namespaces(s int, el *Element, parent map[string]string) (map[string]string, error) {
	scope, copied := parent, false
	kept := el.Attrs[:0:0]
	for _, a := range el.Attrs {
		prefix, isDecl := "", a.Name == "xmlns"
		if strings.HasPrefix(a.Name, "xmlns:") {
			prefix, isDecl = a.Name[6:], true
			if a.Value == "" {
				return nil, p.fail(s, errUndeclare)
			}
		}
		if !isDecl {
			kept = append(kept, a)
			continue
		}
		if !copied {
			scope, copied = make(map[string]string, len(parent)+1), true
			for k, v := range parent {
				scope[k] = v
			}
		}
		scope[prefix] = a.Value
	}
	seen := make(map[string]bool, len(kept))
	for i, a := range kept {
		if j := strings.IndexByte(a.Name, ':'); j >= 0 {
			uri, ok := lookupNS(scope, a.Name[:j])
			if !ok {
				return nil, p.fail(s, errUnbound)
			}
			kept[i].Name = "{" + uri + "}" + a.Name[j+1:]
		}
		if seen[kept[i].Name] {
			return nil, p.fail(s, errDupAttr)
		}
		seen[kept[i].Name] = true
	}
	el.Attrs = kept
	if j := strings.IndexByte(el.Tag, ':'); j >= 0 {
		uri, ok := lookupNS(scope, el.Tag[:j])
		if !ok {
			return nil, p.fail(s, errUnbound)
		}
		el.Tag = "{" + uri + "}" + el.Tag[j+1:]
	} else if uri := scope[""]; uri != "" {
		el.Tag = "{" + uri + "}" + el.Tag
	}
	return scope, nil
}

func lookupNS(scope map[string]string, prefix string) (string, bool) {
	if prefix == "xml" {
		return xmlNS, true
	}
	uri, ok := scope[prefix]
	return uri, ok && uri != ""
}

// endTag parses "</name S?>" from its '<' and closes the open element.
func (p *xparser) endTag(s int) (int, error) {
	i := s + 2
	if p.eof(i) {
		return s, p.fail(s, errUnclosed)
	}
	if !qStart(p.r[i]) {
		return s, p.fail(i, errInvalid)
	}
	e := p.scanName(i) // expat's end-tag scanner takes any colons
	j := e
	for !p.eof(j) && isS(p.r[j]) {
		j++
	}
	if p.eof(j) {
		return s, p.fail(s, errUnclosed)
	}
	if p.r[j] != '>' {
		return s, p.fail(j, errInvalid)
	}
	if top := p.stack[len(p.stack)-1]; string(p.r[i:e]) != top.qname {
		return s, p.fail(i, errMismatch)
	}
	p.stack = p.stack[:len(p.stack)-1]
	return j + 1, nil
}
