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

// SlotValue returns the v attribute of the direct child <p n="name"> of element i, "" when there is no such
// child or it has no v.
func (d *Doc) SlotValue(i int, name string) string {
	for _, c := range d.Children(i) {
		if n, _ := d.elems[c].Attr("n"); n == name && d.elems[c].Name == "p" {
			v, _ := d.elems[c].Attr("v")
			return v
		}
	}
	return ""
}

// SlotPath returns the slot path of element i: the decoded n (UnescapeName) of each <p> from below the root down
// to it, every segment led by "/"; an element with no n, or that is not a <p>, adds no segment.
func (d *Doc) SlotPath(i int) string {
	var parts []string
	for ; i >= 0; i = d.elems[i].Parent {
		if n, ok := d.elems[i].Attr("n"); ok && d.elems[i].Name == "p" {
			parts = append(parts, UnescapeName(n))
		}
	}
	var b strings.Builder
	for k := len(parts) - 1; k >= 0; k-- {
		b.WriteString("/" + parts[k])
	}
	return b.String()
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
			return unmask(sb.String(), d.masked)
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
