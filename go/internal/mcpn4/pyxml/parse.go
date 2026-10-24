package pyxml

import "unicode/utf8"

// Parse is ElementTree.fromstring(text): the root element, or the *Error expat raises.
func Parse(text string) (*Element, error) {
	p := &xparser{r: decodeUTF8(text)}
	p.n = len(p.r)
	if p.at(0) == 0xFEFF {
		p.bom = 1
	}
	if err := p.document(); err != nil {
		return nil, err
	}
	return p.root, nil
}

// decodeUTF8 decodes text, one bad rune per undecodable byte (expat stops at the first).
func decodeUTF8(text string) []rune {
	out := make([]rune, 0, len(text))
	for len(text) > 0 {
		c, size := utf8.DecodeRuneInString(text)
		if c == utf8.RuneError && size == 1 {
			c = bad
		}
		out = append(out, c)
		text = text[size:]
	}
	return out
}

func (p *xparser) document() error {
	i, err := p.prolog()
	if err != nil {
		return err
	}
	if i, err = p.element(i); err != nil {
		return err
	}
	return p.epilog(i)
}
