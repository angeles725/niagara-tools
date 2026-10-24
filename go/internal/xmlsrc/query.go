package xmlsrc

import (
	"bytes"
	"encoding/xml"
	"strings"
)

// Attr returns the value of the attribute named name (prefix included) and whether it is present.
func (e Element) Attr(name string) (string, bool) {
	for _, a := range e.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Find returns the indexes of the elements named name, in document order.
func (d *Doc) Find(name string) []int {
	var out []int
	for i, e := range d.elems {
		if e.Name == name {
			out = append(out, i)
		}
	}
	return out
}

// Children returns the indexes of the direct children of element i, in document order.
func (d *Doc) Children(i int) []int {
	var out []int
	for j := i + 1; j < len(d.elems) && d.elems[j].Depth > d.elems[i].Depth; j++ {
		if d.elems[j].Parent == i {
			out = append(out, j)
		}
	}
	return out
}

// Text returns the character data directly inside element i (not its descendants'), entity-
// decoded, CDATA included. It re-reads the element's content on each call, so only rules that ask
// for text pay for it.
func (d *Doc) Text(i int) string {
	e := d.elems[i]
	dec := xml.NewDecoder(bytes.NewReader(d.data[e.start:e.end]))
	dec.Strict = true
	var sb strings.Builder
	depth := 0
	for {
		tok, err := dec.RawToken()
		if err != nil { // io.EOF: Parse already proved the span well-formed
			return sb.String()
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
		case xml.CharData:
			if depth == 0 {
				sb.Write(t)
			}
		}
	}
}
