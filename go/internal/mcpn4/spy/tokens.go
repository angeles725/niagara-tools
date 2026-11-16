package spy

import (
	"html"
	"strings"
)

// token is one piece of a page: text (name ""), a start tag or an end tag (end true), with the
// attributes of a start tag by lower-case name.
type token struct {
	name  string
	end   bool
	attrs map[string]string
	text  string
}

// rawText are the elements whose content is not page text.
var rawText = map[string]bool{"script": true, "style": true}

// scanner splits a page the way a lenient reader does: tags by their angle brackets (a quoted
// attribute value may hold '>'), comments and declarations dropped, script and style content
// dropped, entities decoded. It never fails: what is not a tag is text. Each byte of the page is
// looked at a bounded number of times, so reading is linear in the page size.
type scanner struct {
	page string
	i    int
}

// next is the next token, ok false at the end of the page.
func (sc *scanner) next() (token, bool) {
	page := sc.page
	for sc.i < len(page) {
		i := sc.i
		lt := strings.IndexByte(page[i:], '<')
		if lt < 0 {
			sc.i = len(page)
			return token{text: html.UnescapeString(page[i:])}, true
		}
		if lt > 0 {
			sc.i = i + lt
			return token{text: html.UnescapeString(page[i : i+lt])}, true
		}
		switch {
		case strings.HasPrefix(page[i:], "<!--"):
			end := strings.Index(page[i:], "-->")
			if end < 0 {
				sc.i = len(page)
				return token{}, false
			}
			sc.i = i + end + 3
			continue
		case i+1 >= len(page) || !isTagStart(page[i+1]):
			sc.i = i + 1
			return token{text: "<"}, true
		}
		end := tagEnd(page, i)
		if end < 0 {
			sc.i = len(page)
			return token{}, false
		}
		t := parseTag(page[i+1 : end])
		sc.i = end + 1
		if t.name == "" {
			continue
		}
		if rawText[t.name] && !t.end {
			sc.i = closingTag(page, sc.i, t.name)
		}
		return t, true
	}
	return token{}, false
}

// closingTag is the index of the first "</name" at or after from, compared without case; the
// end of the page when there is none. It walks the "</" of the page once.
func closingTag(page string, from int, name string) int {
	for i := from; ; {
		at := strings.Index(page[i:], "</")
		if at < 0 {
			return len(page)
		}
		i += at
		if rest := page[i+2:]; len(rest) >= len(name) && strings.EqualFold(rest[:len(name)], name) {
			return i
		}
		i += 2
	}
}

func isTagStart(c byte) bool {
	return c == '/' || c == '!' || c == '?' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// tagEnd is the index of the '>' that closes the tag opened at start, -1 when it never closes.
func tagEnd(page string, start int) int {
	quote := byte(0)
	for i := start + 1; i < len(page); i++ {
		c := page[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return i
		}
	}
	return -1
}

// parseTag reads the inside of a tag; a declaration (<!doctype, <?xml) has no name.
func parseTag(inner string) token {
	inner = strings.TrimSuffix(strings.TrimSpace(inner), "/")
	if inner == "" || inner[0] == '!' || inner[0] == '?' {
		return token{}
	}
	t := token{attrs: map[string]string{}}
	if inner[0] == '/' {
		t.end, inner = true, inner[1:]
	}
	nameEnd := strings.IndexAny(inner, " \t\r\n")
	if nameEnd < 0 {
		nameEnd = len(inner)
	}
	t.name = strings.ToLower(inner[:nameEnd])
	rest := inner[nameEnd:]
	for {
		rest = strings.TrimLeft(rest, " \t\r\n")
		if rest == "" {
			return t
		}
		eq := strings.IndexAny(rest, "= \t\r\n")
		if eq < 0 || rest[eq] != '=' {
			if eq < 0 {
				eq = len(rest)
			}
			t.attrs[strings.ToLower(rest[:eq])] = ""
			rest = rest[eq:]
			continue
		}
		key, value := strings.ToLower(rest[:eq]), rest[eq+1:]
		switch {
		case value != "" && (value[0] == '"' || value[0] == '\''):
			close := strings.IndexByte(value[1:], value[0])
			if close < 0 {
				close = len(value) - 1
			}
			t.attrs[key], rest = html.UnescapeString(value[1:1+close]), value[min(len(value), close+2):]
		default:
			end := strings.IndexAny(value, " \t\r\n")
			if end < 0 {
				end = len(value)
			}
			t.attrs[key], rest = html.UnescapeString(value[:end]), value[end:]
		}
	}
}
