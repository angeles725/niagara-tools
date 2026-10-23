package xmlsrc

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
)

// run tokenizes data once with encoding/xml in strict mode (entities, character references,
// character ranges and UTF-8 are its checks).
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
			if len(stack) == 0 {
				return nil, p.fail(off, "end tag </%s> without a start tag", qname(t.Name))
			}
			elems[stack[len(stack)-1]].end = off
			stack = stack[:len(stack)-1]
		}
	}
	if !rooted {
		return nil, p.fail(int64(len(data)), "no root element")
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

// syntaxMsg drops encoding/xml's "XML syntax error on line N: " prefix; the position is ours.
func syntaxMsg(err error) string {
	var se *xml.SyntaxError
	if errors.As(err, &se) {
		return se.Msg
	}
	return err.Error()
}
