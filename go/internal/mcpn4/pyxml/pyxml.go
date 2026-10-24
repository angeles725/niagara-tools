// Package pyxml is the XML parser of files.parse_px: xml.etree.ElementTree.fromstring(str) on
// the target CPython, which feeds the text to expat as UTF-8 (any declared encoding is ignored)
// with namespace processing on. It reproduces expat's well-formedness verdict, its error text
// ("<message>: line L, column C", token-start positions, columns in characters) and the element
// tree ElementTree builds: "{uri}local" names, xmlns declarations removed, attribute values
// normalized.
//
// Ported from niagara-go internal/pyxml (b291c25c), whose testdata/xmlerrors.json holds expat's
// verdicts on 3,000+ documents, in its ElementTree mode with the declared encoding dropped. A
// DOCTYPE is not read: parse_px refuses one before parsing, so Parse answers ErrDoctype there.
package pyxml

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Element is an ElementTree element: Tag is "{uri}local" for a namespaced name.
type Element struct {
	Tag      string
	Attrs    []Attr // document order, xmlns declarations removed
	Children []*Element
}

// Attr is one attribute: Name is "{uri}local" for a namespaced name.
type Attr struct{ Name, Value string }

// Get is Element.get(name): the value and whether the attribute exists.
func (e *Element) Get(name string) (string, bool) {
	for _, a := range e.Attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return "", false
}

// Iter is Element.iter(): e and every element below it, depth first in document order. It
// keeps its own stack, so a deeply nested document needs no deep recursion.
func (e *Element) Iter() []*Element {
	var out []*Element
	for stack := []*Element{e}; len(stack) > 0; {
		el := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out = append(out, el)
		for i := len(el.Children) - 1; i >= 0; i-- {
			stack = append(stack, el.Children[i])
		}
	}
	return out
}

// Error is ElementTree's ParseError str(); Line is 1-based, Col 0-based.
type Error struct {
	Msg       string
	Line, Col int
}

func (e *Error) Error() string { return fmt.Sprintf("%s: line %d, column %d", e.Msg, e.Line, e.Col) }

// ErrDoctype answers a document whose prolog holds a DOCTYPE, which this port does not read.
var ErrDoctype = errors.New("pyxml: a DOCTYPE is not supported")

const (
	errNoElement  = "no element found"
	errInvalid    = "not well-formed (invalid token)"
	errUnclosed   = "unclosed token"
	errMismatch   = "mismatched tag"
	errDupAttr    = "duplicate attribute"
	errJunk       = "junk after document element"
	errUndefined  = "undefined entity"
	errBadCharRef = "reference to invalid character number"
	errCDATA      = "unclosed CDATA section"
	errMisplaced  = "XML or text declaration not at start of entity"
	errBadDecl    = "XML declaration not well-formed"
	errSyntax     = "syntax error"
	errUnbound    = "unbound prefix"
	errUndeclare  = "must not undeclare prefix"
	errPERef      = "illegal parameter entity reference"
	bad           = -1 // an undecodable byte in the rune slice
)

const xmlNS = "http://www.w3.org/XML/1998/namespace"

var predefined = map[string]rune{"lt": '<', "gt": '>', "amp": '&', "apos": '\'', "quot": '"'}

type xparser struct {
	r     []rune
	n     int
	stack []*openElem
	root  *Element
	bom   int // runes the byte order mark occupies (it counts as that many columns)
}

type openElem struct {
	qname string
	elem  *Element
	ns    map[string]string // prefix -> uri in scope ("" is the default namespace)
}

func (p *xparser) at(i int) rune {
	if i < p.n {
		return p.r[i]
	}
	return bad - 1 // end of input
}

func (p *xparser) eof(i int) bool { return i >= p.n }

// fail builds the error at rune index pos: expat counts lines at LF, CR and CRLF and columns
// in characters from the last line break.
func (p *xparser) fail(pos int, msg string) error {
	line, col := 1, 0
	for i := 0; i < pos && i < p.n; i++ {
		switch p.r[i] {
		case '\n':
			line, col = line+1, 0
		case '\r':
			line, col = line+1, 0
			if i+1 < pos && p.at(i+1) == '\n' {
				i++
			}
		default:
			col++
		}
	}
	return &Error{Msg: msg, Line: line, Col: col}
}

// failEOF is "no element found" at the end of input. Inside the root element a trailing CR
// is still a pending token (it may start a CRLF), so expat reports the position before it.
func (p *xparser) failEOF() error {
	pos := p.n
	if len(p.stack) > 0 && pos > 0 && p.r[pos-1] == '\r' {
		pos--
	}
	return p.fail(pos, errNoElement)
}

func isXMLChar(c rune) bool {
	return c == 0x9 || c == 0xA || c == 0xD || 0x20 <= c && c <= 0xD7FF ||
		0xE000 <= c && c <= 0xFFFD || 0x10000 <= c && c <= 0x10FFFF
}

func isS(c rune) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// isNameStart and isNameChar follow XML 1.0 (4th edition) as expat does: BMP only.
func isNameStart(c rune) bool {
	if c < utf8.RuneSelf {
		return c == '_' || c == ':' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
	}
	return c <= 0xFFFF && (unicode.IsLetter(c) && !unicode.Is(unicode.Lm, c) || unicode.Is(unicode.Nl, c))
}

func isNameChar(c rune) bool {
	if c < utf8.RuneSelf {
		return isNameStart(c) || c == '-' || c == '.' || '0' <= c && c <= '9'
	}
	return isNameStart(c) || c <= 0xFFFF && (c == 0xB7 || unicode.In(c, unicode.Mn, unicode.Mc, unicode.Me, unicode.Nd, unicode.Lm))
}

// scanName returns the end of the name chars starting at i (colons allowed).
func (p *xparser) scanName(i int) int {
	for i < p.n && isNameChar(p.r[i]) {
		i++
	}
	return i
}

// qStart reports a character that may start an element, attribute, PI or entity name: with
// namespace processing expat does not let a name start with ':'.
func qStart(c rune) bool { return isNameStart(c) && c != ':' }

// scanNC scans a PI target or entity name: with namespace processing it stops at ':'.
func (p *xparser) scanNC(i int) int {
	for i < p.n && isNameChar(p.r[i]) && p.r[i] != ':' {
		i++
	}
	return i
}

// scanQName scans an element or attribute name: at most one ':', followed by a name start
// character.
func (p *xparser) scanQName(i int) (int, error) {
	e := p.scanNC(i)
	if p.at(e) != ':' {
		return e, nil
	}
	if p.eof(e + 1) {
		return e + 1, nil
	}
	if !qStart(p.r[e+1]) {
		return e, p.fail(e+1, errInvalid)
	}
	if e = p.scanNC(e + 1); p.at(e) == ':' {
		return e, p.fail(e, errInvalid)
	}
	return e, nil
}
