package xmlsrc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

// run tokenizes data once with encoding/xml in strict mode (entities, character references,
// character ranges and UTF-8 are its checks) and adds the document-level checks RawToken leaves out:
// matching end tags, one root, no text outside it, unique attributes and the XML declaration first.
// A DOCTYPE is refused: Niagara never writes one, and its entities would need a DTD reader.
func (p *parser) run(data []byte) ([]Element, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	var elems []Element
	var stack []int
	rooted := false
	for {
		off := dec.InputOffset()
		tok, err := dec.RawToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, p.fail(dec.InputOffset(), "%s", syntaxMsg(err))
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if len(stack) == 0 && rooted {
				return nil, p.fail(off, "junk after document element")
			}
			e, err := p.element(t, off, dec.InputOffset())
			if err != nil {
				return nil, err
			}
			e.Depth, e.Parent = len(stack), -1
			if len(stack) > 0 {
				e.Parent = stack[len(stack)-1]
			}
			rooted = true
			stack = append(stack, len(elems))
			elems = append(elems, e)
		case xml.EndElement:
			name := qname(t.Name)
			if len(stack) == 0 {
				return nil, p.fail(off, "end tag </%s> without a start tag", name)
			}
			top := &elems[stack[len(stack)-1]]
			if name != top.Name {
				return nil, p.fail(off, "element <%s> opened at %d:%d closed by </%s>", top.Name, top.Line, top.Col, name)
			}
			top.end = off
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 && !isSpace(t) {
				lead := len(t) - len(bytes.TrimLeft(t, " \t\r\n"))
				if rooted {
					return nil, p.fail(off+int64(lead), "junk after document element")
				}
				return nil, p.fail(off+int64(lead), "text outside the root element")
			}
		case xml.ProcInst:
			if t.Target == "xml" && off != 0 {
				return nil, p.fail(off, "XML declaration not at the start of the document")
			}
		case xml.Directive:
			return nil, p.fail(off, "<!%s ...> is not supported", firstWord(t))
		}
	}
	end := int64(len(data))
	if len(stack) > 0 {
		top := elems[stack[len(stack)-1]]
		return nil, p.fail(end, "element <%s> opened at %d:%d is never closed", top.Name, top.Line, top.Col)
	}
	if !rooted {
		return nil, p.fail(end, "no root element")
	}
	return elems, nil
}

func (p *parser) element(t xml.StartElement, off, contentStart int64) (Element, error) {
	line, col := p.pos.at(off)
	e := Element{Name: qname(t.Name), Line: line, Col: col, start: contentStart, end: contentStart}
	if len(t.Attr) > 0 {
		e.Attrs = make([]Attr, len(t.Attr))
	}
	for i, a := range t.Attr {
		e.Attrs[i] = Attr{Name: qname(a.Name), Value: a.Value}
		for _, prev := range e.Attrs[:i] {
			if prev.Name == e.Attrs[i].Name {
				return Element{}, p.fail(off, "duplicate attribute %s", prev.Name)
			}
		}
	}
	return e, nil
}

// qname is a name as written: RawToken keeps the prefix in Space.
func qname(n xml.Name) string {
	if n.Space == "" {
		return n.Local
	}
	return n.Space + ":" + n.Local
}

func firstWord(d xml.Directive) string {
	if i := bytes.IndexAny(d, " \t\r\n"); i >= 0 {
		return string(d[:i])
	}
	return string(d)
}

// syntaxMsg drops encoding/xml's "XML syntax error on line N: " prefix; the position is ours.
func syntaxMsg(err error) string {
	var se *xml.SyntaxError
	if errors.As(err, &se) {
		return se.Msg
	}
	return err.Error()
}
