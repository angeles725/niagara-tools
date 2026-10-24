// Entity and character references of the expat-compatible parser (see pyxml.go).

package pyxml

import "strings"

// rawAttr is one attribute as scanned: its name, where it starts and its value's runes.
type rawAttr struct {
	name   string
	pos    int // index of the name
	vs, ve int // value runes between the quotes
}

// refSyntax checks the reference at '&' a and returns the index after its ';'; a reference
// cut by the end of input is an unclosed token starting at partial.
func (p *xparser) refSyntax(a, partial int) (int, error) {
	i := a + 1
	if p.eof(i) {
		return a, p.fail(partial, errUnclosed)
	}
	if p.r[i] == '#' {
		i++
		hex := p.at(i) == 'x'
		if hex {
			i++
		}
		for first := true; ; i, first = i+1, false {
			if p.eof(i) {
				return a, p.fail(partial, errUnclosed)
			}
			if c := p.r[i]; c == ';' && !first {
				return i + 1, nil
			} else if !isDigit(c, hex) {
				return a, p.fail(i, errInvalid)
			}
		}
	}
	if !qStart(p.r[i]) {
		return a, p.fail(i, errInvalid)
	}
	i = p.scanNC(i)
	if p.eof(i) {
		return a, p.fail(partial, errUnclosed)
	}
	if p.r[i] != ';' {
		return a, p.fail(i, errInvalid)
	}
	return i + 1, nil
}

func isDigit(c rune, hex bool) bool {
	return '0' <= c && c <= '9' || hex && ('a' <= c && c <= 'f' || 'A' <= c && c <= 'F')
}

const (
	refOK = iota
	refUndefined
	refBadNumber
)

// refValue resolves the syntactically valid reference r[a:e] (e is after the ';').
func (p *xparser) refValue(a, e int) (string, int) {
	body := p.r[a+1 : e-1]
	if body[0] != '#' {
		name := string(body)
		if v, ok := predefined[name]; ok {
			return string(v), refOK
		}
		return "", refUndefined
	}
	base, digits := 10, body[1:]
	if digits[0] == 'x' {
		base, digits = 16, digits[1:]
	}
	v := 0
	for _, c := range digits {
		d := int(c - '0')
		if c > '9' {
			d = int(c|0x20-'a') + 10
		}
		if v = v*base + d; v > 0x10FFFF {
			return "", refBadNumber
		}
	}
	if !isXMLChar(rune(v)) {
		return "", refBadNumber
	}
	return string(rune(v)), refOK
}

// contentRef handles a reference in character data.
func (p *xparser) contentRef(a int) (int, error) {
	e, err := p.refSyntax(a, a)
	if err != nil {
		return a, err
	}
	switch _, kind := p.refValue(a, e); {
	case kind == refBadNumber:
		return a, p.fail(a, errBadCharRef)
	case kind == refUndefined:
		return a, p.fail(a, errUndefined)
	}
	return e, nil
}

// attrValue normalises an attribute value as expat does for CDATA attributes.
func (p *xparser) attrValue(s int, a rawAttr) (string, error) {
	var b strings.Builder
	for i := a.vs; i < a.ve; i++ {
		switch c := p.r[i]; c {
		case '&':
			e, _ := p.refSyntax(i, s)
			v, kind := p.refValue(i, e)
			switch {
			case kind == refBadNumber:
				return "", p.fail(i, errBadCharRef)
			case kind == refUndefined:
				return "", p.fail(s, errUndefined)
			}
			b.WriteString(v)
			i = e - 1
		case '\r':
			if p.at(i+1) == '\n' {
				i++
			}
			b.WriteByte(' ')
		case '\n', '\t':
			b.WriteByte(' ')
		default:
			b.WriteRune(c)
		}
	}
	return b.String(), nil
}
