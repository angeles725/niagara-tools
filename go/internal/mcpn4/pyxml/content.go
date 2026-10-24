// The element content of the expat-compatible parser (see pyxml.go): character data, comments,
// CDATA sections and processing instructions.

package pyxml

// element parses the root element starting at its '<' and returns the index after it.
func (p *xparser) element(i int) (int, error) {
	var err error
	if i, err = p.startTag(i); err != nil || len(p.stack) == 0 {
		return i, err
	}
	for {
		if p.eof(i) {
			return i, p.failEOF()
		}
		switch c := p.r[i]; {
		case c == '<':
			if i, err = p.markup(i); err != nil || len(p.stack) == 0 {
				return i, err
			}
		case c == '&':
			if i, err = p.contentRef(i); err != nil {
				return i, err
			}
		case c == ']' && p.at(i+1) == ']' && p.at(i+2) == '>':
			return i, p.fail(i+2, errInvalid)
		case c == bad || !isXMLChar(c):
			return i, p.fail(i, errInvalid)
		default:
			i++
		}
	}
}

// markup is a '<' inside the root element.
func (p *xparser) markup(s int) (int, error) {
	switch c := p.at(s + 1); {
	case p.eof(s + 1):
		return s, p.fail(s, errUnclosed)
	case c == '/':
		return p.endTag(s)
	case c == '!':
		return p.contentBang(s)
	case c == '?':
		return p.pi(s, false)
	case qStart(c):
		return p.startTag(s)
	default:
		return s, p.fail(s+1, errInvalid)
	}
}

func (p *xparser) contentBang(s int) (int, error) {
	i := s + 2
	switch {
	case p.eof(i):
		return s, p.fail(s, errUnclosed)
	case p.r[i] == '-':
		return p.comment(s)
	case p.r[i] != '[':
		return s, p.fail(i, errInvalid)
	}
	for _, c := range "[CDATA[" {
		if p.eof(i) {
			return s, p.fail(s, errUnclosed)
		}
		if p.r[i] != c {
			return s, p.fail(i, errInvalid)
		}
		i++
	}
	i0 := i
	for ; !p.eof(i); i++ {
		c := p.r[i]
		if c == ']' && p.at(i+1) == ']' && p.at(i+2) == '>' {
			return i + 3, nil
		}
		if c == bad || !isXMLChar(c) {
			return s, p.fail(i, errInvalid)
		}
	}
	// a trailing CR or "]"/"]]" is a pending token: expat reports the position before it
	pos := p.n
	if p.r[pos-1] == '\r' {
		pos--
	} else {
		for k := 0; k < 2 && pos > i0 && p.r[pos-1] == ']'; k++ {
			pos--
		}
	}
	return s, p.fail(pos, errCDATA)
}

// comment scans "<!--...-->" from its '<'.
func (p *xparser) comment(s int) (int, error) {
	i := s + 3
	if p.eof(i) {
		return s, p.fail(s, errUnclosed)
	}
	if p.r[i] != '-' {
		return s, p.fail(i, errInvalid)
	}
	for i++; !p.eof(i); i++ {
		c := p.r[i]
		if c == bad || !isXMLChar(c) {
			return s, p.fail(i, errInvalid)
		}
		if c == '-' && p.at(i+1) == '-' {
			switch {
			case p.eof(i + 2):
				return s, p.fail(s, errUnclosed)
			case p.r[i+2] != '>':
				return s, p.fail(i+2, errInvalid)
			}
			return i + 3, nil
		}
	}
	return s, p.fail(s, errUnclosed)
}

// pi scans "<?target ...?>" from its '<'; atStart allows the XML declaration.
func (p *xparser) pi(s int, atStart bool) (int, error) {
	i := s + 2
	if p.eof(i) {
		return s, p.fail(s, errUnclosed)
	}
	if !qStart(p.r[i]) {
		return s, p.fail(i, errInvalid)
	}
	e := p.scanNC(i)
	target := string(p.r[i:e])
	if len(target) == 3 && target != "xml" && (target[0]|0x20) == 'x' && (target[1]|0x20) == 'm' && (target[2]|0x20) == 'l' {
		return s, p.fail(e, errInvalid)
	}
	body := e
	switch c := p.at(e); {
	case p.eof(e):
		return s, p.fail(s, errUnclosed)
	case c == '?':
		if p.eof(e + 1) {
			return s, p.fail(s, errUnclosed)
		}
		if p.r[e+1] != '>' {
			return s, p.fail(e+1, errInvalid)
		}
		e += 2
	case !isS(c):
		return s, p.fail(e, errInvalid)
	default:
		for e++; ; e++ {
			if p.eof(e) {
				return s, p.fail(s, errUnclosed)
			}
			if c := p.r[e]; c == bad || !isXMLChar(c) {
				return s, p.fail(e, errInvalid)
			}
			if p.r[e] == '?' && p.at(e+1) == '>' {
				e += 2
				break
			}
		}
	}
	if target == "xml" {
		if !atStart {
			return s, p.fail(s, errMisplaced)
		}
		return e, p.xmlDecl(body, e-2)
	}
	return e, nil
}
