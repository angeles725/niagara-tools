// Package xmlsrc is the XML scanner the kit's XML and .bog lints share: each file is read once,
// checked for well-formedness and turned into a flat, ordered element list. A malformed document is
// an *Error with path:line:col, never an empty result. Writing XML and XPath are out of scope.
// Design: R8-D12 (go/README.md, "XML rules").
package xmlsrc

import (
	"bytes"
	"fmt"
)

// Attr is one attribute as written: Name keeps its prefix ("xmlns:p"), Value is entity-decoded.
type Attr struct{ Name, Value string }

// Element is one start tag. Line and Col are 1-based and point at its '<'; Col counts bytes.
type Element struct {
	Name   string // as written, prefix kept ("p:a")
	Attrs  []Attr // in document order
	Line   int
	Col    int
	Depth  int  // 0 for the root
	Parent int  // index of the parent in Elements(); -1 for the root
	Empty  bool // written as an empty-element tag, <name .../>, not <name ...></name>

	start, end int64 // byte span of the content, for Text
}

// Doc is one well-formed document.
type Doc struct {
	Path   string
	data   []byte
	elems  []Element
	warns  []*Error
	masked bool // data holds nulMark for U+0000 (nul.go)
}

// Warnings are what Doc reads although XML 1.0 forbids it, because Niagara reads it: at most one per kind, at
// its first occurrence. Every tool that reads a .bog prints them; none is silent.
func (d *Doc) Warnings() []*Error { return d.warns }

// Elements returns every element in document order; index i is the i-th start tag.
func (d *Doc) Elements() []Element { return d.elems }

// Open reads path once and parses it; a zipped .bog is parsed from its file.xml member, with
// positions inside that member. A read error is returned as the os package words it.
func Open(path string) (*Doc, error) {
	data, err := readSource(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

// Parse checks data for well-formedness and builds its Doc; path only labels errors.
func Parse(path string, data []byte) (*Doc, error) {
	p := &parser{path: path, pos: newPositions(data)}
	text, err := p.toUTF8(data)
	if err != nil {
		return nil, err
	}
	if len(text) != len(data) {
		p.pos = newPositions(text)
	}
	if err := p.mask(text); err != nil {
		return nil, err
	}
	elems, err := p.run(p.text)
	if warn, err := p.nulVerdict(err); err != nil {
		return nil, err
	} else if warn != nil {
		return &Doc{Path: path, data: p.text, elems: elems, warns: []*Error{warn}, masked: true}, nil
	}
	return &Doc{Path: path, data: p.text, elems: elems}, nil
}

type parser struct {
	path  string
	pos   positions // of the text as written; at maps an offset of the masked text onto it
	text  []byte    // what run reads: the text with U+0000 masked (nul.go)
	marks []mark
	root  string // the first start tag's name
}

func (p *parser) fail(off int64, format string, args ...any) error {
	return p.failAt(p.unmasked(off), format, args...)
}

// failAt is fail for an offset of the text as written.
func (p *parser) failAt(off int64, format string, args ...any) *Error {
	line, col := p.pos.at(off)
	return &Error{Path: p.path, Line: line, Col: col, Msg: fmt.Sprintf(format, args...)}
}

func isSpace(b []byte) bool { return len(bytes.TrimLeft(b, " \t\r\n")) == 0 }
