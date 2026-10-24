// Prolog, epilog and XML declaration of the expat-compatible parser (see pyxml.go). Outside
// the root element expat tokenizes with its DTD tokenizer: a token that is valid but not
// allowed there is a syntax error before the root and "junk after document element" after it.

package pyxml

import (
	"strings"
	"unicode/utf8"
)

// prolog skips everything before the root element and returns the index of its '<'.
func (p *xparser) prolog() (int, error) {
	start := p.bom // the byte order mark is skipped but counts as columns
	for i := start; ; {
		if p.eof(i) {
			return i, p.failEOF()
		}
		c := p.r[i]
		var err error
		switch {
		case isS(c):
			i++
			continue
		case c != '<':
			e, kind, err := p.prologToken(i)
			if err != nil {
				return i, err
			}
			if kind == tokPERef {
				return e, p.fail(i, errPERef)
			}
			return e, p.fail(i, errSyntax)
		}
		switch c1 := p.at(i + 1); {
		case p.eof(i + 1):
			return i, p.fail(i, errUnclosed)
		case c1 == '?':
			i, err = p.pi(i, i == start)
		case c1 == '!':
			var isDoctype bool
			if i, isDoctype, err = p.bang(i, true); err == nil && isDoctype {
				return i, ErrDoctype
			}
		case isInstanceStart(c1):
			return i, nil
		default:
			return i, p.fail(i+1, errInvalid)
		}
		if err != nil {
			return i, err
		}
	}
}

// epilog allows only white space, comments and processing instructions after the root.
func (p *xparser) epilog(i int) error {
	for !p.eof(i) {
		c := p.r[i]
		var err error
		switch {
		case isS(c):
			i++
			continue
		case c != '<':
			if _, _, err := p.prologToken(i); err != nil {
				return err
			}
			return p.fail(i, errJunk)
		}
		switch c1 := p.at(i + 1); {
		case p.eof(i + 1):
			return p.fail(i, errUnclosed)
		case c1 == '?':
			// expat 2.9 reads an XML declaration here as junk, not as a misplaced one
			if i, err = p.pi(i, false); err != nil && err.(*Error).Msg == errMisplaced {
				return p.fail(i, errJunk)
			}
		case c1 == '!':
			i, _, err = p.bang(i, false)
		case isInstanceStart(c1):
			return p.fail(i, errJunk)
		default:
			return p.fail(i+1, errInvalid)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// bang handles "<!" outside the root: a comment is skipped; "<!DOCTYPE" returns isDoctype
// with i after the keyword when allowed; any other declaration is not allowed here.
func (p *xparser) bang(s int, doctypeOK bool) (int, bool, error) {
	j := s + 2
	if p.eof(j) {
		return s, false, p.fail(s, errUnclosed)
	}
	switch c := p.r[j]; {
	case c == '-':
		i, err := p.comment(s)
		return i, false, err
	case c == '[':
		return s, false, p.notAllowed(s)
	case !isKeywordChar(c):
		return s, false, p.fail(j, errInvalid)
	}
	e := p.scanKeyword(j)
	if p.eof(e) {
		return s, false, p.fail(s, errUnclosed)
	}
	if c := p.r[e]; !isS(c) && c != '%' {
		return s, false, p.fail(e, errInvalid)
	}
	if doctypeOK && string(p.r[j:e]) == "DOCTYPE" {
		return e, true, nil
	}
	return s, false, p.notAllowed(s)
}

// isInstanceStart is a character after '<' that expat's DTD tokenizer reads as the start of an
// element: a name start or any non-ASCII character (its name is checked as the tag is read).
func isInstanceStart(c rune) bool { return qStart(c) || c >= utf8.RuneSelf }

// isKeywordChar is a character of a declaration keyword: expat's scanDecl takes ASCII
// letters and '_' only (a ':' is its own byte type under namespace processing).
func isKeywordChar(c rune) bool { return c < utf8.RuneSelf && qStart(c) }

// scanKeyword scans a declaration keyword.
func (p *xparser) scanKeyword(i int) int {
	for i < p.n && isKeywordChar(p.r[i]) {
		i++
	}
	return i
}

// notAllowed is the error for a valid token at s that cannot appear outside the root.
func (p *xparser) notAllowed(s int) error {
	if p.root == nil {
		return p.fail(s, errSyntax)
	}
	return p.fail(s, errJunk)
}

const (
	tokOther = iota
	tokPERef
)

// prologToken scans one DTD token at i (not white space, not '<'): an invalid or cut token is
// returned as the error, a valid one as its end and kind.
func (p *xparser) prologToken(i int) (int, int, error) {
	c := p.r[i]
	switch {
	case c == bad || !isXMLChar(c):
		return i, 0, p.fail(i, errInvalid)
	case isNameChar(c):
		return p.prologName(i)
	case c == '"' || c == '\'':
		e := i + 1
		for ; !p.eof(e) && p.r[e] != c; e++ {
			if d := p.r[e]; d == bad || !isXMLChar(d) {
				return i, 0, p.fail(e, errInvalid)
			}
		}
		if p.eof(e) {
			return i, 0, p.fail(i, errUnclosed)
		}
		return p.afterName(e+1, ">%[", false)
	case c == '#' || c == '%':
		if p.eof(i + 1) {
			return i, 0, p.fail(i, errUnclosed)
		}
		if c == '%' && isS(p.r[i+1]) {
			return i + 1, tokOther, nil
		}
		if !qStart(p.r[i+1]) {
			return i, 0, p.fail(i+1, errInvalid)
		}
		e := p.scanNC(i + 1)
		if c == '#' {
			return p.afterName(e, ")|>%,", false)
		}
		if p.eof(e) {
			return i, 0, p.fail(i, errUnclosed)
		}
		if p.r[e] != ';' {
			return i, 0, p.fail(e, errInvalid)
		}
		return e + 1, tokPERef, nil
	case c == ')':
		return p.afterName(i+1, ">,|)", true)
	case strings.ContainsRune("[]>(|,", c):
		return i + 1, tokOther, nil
	}
	return i, 0, p.fail(i, errInvalid)
}

// prologName scans a name token at i. Under namespace processing a ':' makes a name prefixed;
// a ':' at the start, a second one or one not followed by a name character makes it an NMTOKEN,
// which cannot take an occurrence indicator; a name cut right after its ':' is unclosed.
func (p *xparser) prologName(i int) (int, int, error) {
	nmtoken, prefixed := !qStart(p.r[i]), false
	e := i + 1
	for ; e < p.n && isNameChar(p.r[e]); e++ {
		switch {
		case p.r[e] != ':' || nmtoken:
		case prefixed:
			nmtoken = true
		case p.eof(e + 1):
			return i, 0, p.fail(i, errUnclosed)
		case p.r[e+1] != ':' && isNameChar(p.r[e+1]):
			prefixed = true
		default:
			nmtoken = true
		}
	}
	return p.afterName(e, ">)[%,|", !nmtoken)
}

// afterName checks the character ending a name-like token at e: white space, the end of
// input, one of ends, or (with occurrence) one of "*?+".
func (p *xparser) afterName(e int, ends string, occurrence bool) (int, int, error) {
	if p.eof(e) {
		return e, tokOther, nil
	}
	c := p.r[e]
	switch {
	case isS(c) || strings.ContainsRune(ends, c):
		return e, tokOther, nil
	case occurrence && strings.ContainsRune("*?+", c):
		return e + 1, tokOther, nil
	}
	return e, 0, p.fail(e, errInvalid)
}
