// The XML declaration of the expat-compatible parser (see pyxml.go): expat's doParseXmlDecl.

package pyxml

import "strings"

// xmlDecl is expat's doParseXmlDecl over the declaration body r[b:end] ("?>" excluded).
func (p *xparser) xmlDecl(b, end int) error {
	i := b
	name, val, ok := p.pseudoAttr(&i, end)
	if !ok || name < 0 {
		return p.fail(i, errBadDecl)
	}
	if !p.asciiIs(name, "version") {
		return p.fail(name, errBadDecl)
	}
	if !p.versionOK(val, i-1) {
		return p.fail(val, errBadDecl)
	}
	if name, val, ok = p.pseudoAttr(&i, end); !ok {
		return p.fail(i, errBadDecl)
	}
	if name < 0 {
		return nil
	}
	if p.asciiIs(name, "encoding") {
		if c := p.ascii(val, end); !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z') {
			return p.fail(val, errBadDecl)
		}
		if name, val, ok = p.pseudoAttr(&i, end); !ok {
			return p.fail(i, errBadDecl)
		}
		if name < 0 {
			return nil
		}
	}
	if !p.asciiIs(name, "standalone") {
		return p.fail(name, errBadDecl)
	}
	if v := string(p.r[val : i-1]); v != "yes" && v != "no" {
		return p.fail(val, errBadDecl)
	}
	for isS(p.ascii(i, end)) {
		i++
	}
	if i != end {
		return p.fail(i, errBadDecl)
	}
	return nil
}

// ascii is expat's toAscii: the character at i, -1 for non-ASCII or at end.
func (p *xparser) ascii(i, end int) rune {
	if i >= end || p.r[i] < 0 || p.r[i] >= 0x80 {
		return -1
	}
	return p.r[i]
}

// asciiIs reports whether the name starting at i is exactly kw (it ends at '=' or space).
func (p *xparser) asciiIs(i int, kw string) bool {
	for _, c := range kw {
		if p.at(i) != c {
			return false
		}
		i++
	}
	c := p.at(i)
	return c == '=' || isS(c)
}

// versionOK is expat's check of the version value: "1." followed by digits.
func (p *xparser) versionOK(v, e int) bool {
	s := string(p.r[v:e])
	if len(s) < 3 || s[:2] != "1." {
		return false
	}
	return strings.Trim(s[2:], "0123456789") == ""
}

// pseudoAttr is expat's parsePseudoAttribute: name < 0 means none left; on failure *i is the
// bad position.
func (p *xparser) pseudoAttr(i *int, end int) (name, val int, ok bool) {
	ptr := *i
	if ptr == end {
		return -1, 0, true
	}
	if !isS(p.ascii(ptr, end)) {
		return 0, 0, false
	}
	for ptr++; isS(p.ascii(ptr, end)); ptr++ {
	}
	if ptr == end {
		*i = ptr
		return -1, 0, true
	}
	name = ptr
	for {
		c := p.ascii(ptr, end)
		if c == -1 {
			*i = ptr
			return 0, 0, false
		}
		if c == '=' {
			break
		}
		if isS(c) {
			for ptr++; isS(p.ascii(ptr, end)); ptr++ {
			}
			if p.ascii(ptr, end) != '=' {
				*i = ptr
				return 0, 0, false
			}
			break
		}
		ptr++
	}
	if ptr == name {
		*i = ptr
		return 0, 0, false
	}
	for ptr++; isS(p.ascii(ptr, end)); ptr++ {
	}
	q := p.ascii(ptr, end)
	if q != '"' && q != '\'' {
		*i = ptr
		return 0, 0, false
	}
	val = ptr + 1
	for ptr = val; ; ptr++ {
		c := p.ascii(ptr, end)
		if c == q {
			break
		}
		if !('a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '.' || c == '-' || c == '_') {
			*i = ptr
			return 0, 0, false
		}
	}
	*i = ptr + 1
	return name, val, true
}
