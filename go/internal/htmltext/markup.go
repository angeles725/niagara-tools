package htmltext

import (
	"strings"
	"unicode/utf8"
)

// DropMarkup is the text of an HTML page as three Python re.sub passes leave it, each match replaced by
// the newlines it held so line numbers stay: (?is)<(script|style)\b.*?</\1\s*>, then (?s)<!--.*?-->, then
// (?s)<[^>]*>. Entities are left for Unescape. The tag names match in ASCII case only: Python's IGNORECASE
// also takes "<SCRİPT>" (U+0130) for a script element. Each of elements (lower-case names, e.g. "pre") is
// dropped with its content like a script element, after the comments and before the tags.
func DropMarkup(s string, elements ...string) string {
	s = replace(s, func(s string, i int) int { return element(s, i, "script", "style") })
	s = replace(s, func(s string, i int) int { return closeAfter(s, i, "<!--", "-->") })
	if len(elements) > 0 {
		s = replace(s, func(s string, i int) int { return element(s, i, elements...) })
	}
	return replace(s, func(s string, i int) int { return closeAfter(s, i, "<", ">") })
}

// replace runs a leftmost, non-overlapping match: end(s, i) is the end of the match starting at i, or -1.
func replace(s string, end func(s string, i int) int) string {
	var b strings.Builder
	last := 0
	for i := strings.IndexByte(s, '<'); i >= 0; {
		j := end(s, i)
		if j < 0 {
			if k := strings.IndexByte(s[i+1:], '<'); k >= 0 {
				i += 1 + k
				continue
			}
			break
		}
		b.WriteString(s[last:i])
		b.WriteString(strings.Repeat("\n", strings.Count(s[i:j], "\n")))
		last = j
		k := strings.IndexByte(s[j:], '<')
		if k < 0 {
			break
		}
		i = j + k
	}
	if last == 0 {
		return s
	}
	return b.String() + s[last:]
}

// closeAfter is the end of open...close (lazy) at i, or -1.
func closeAfter(s string, i int, open, close string) int {
	if !strings.HasPrefix(s[i:], open) {
		return -1
	}
	if k := strings.Index(s[i+len(open):], close); k >= 0 {
		return i + len(open) + k + len(close)
	}
	return -1
}

// asciiFold reports whether s is the lower-case ASCII name in any ASCII case.
func asciiFold(s, name string) bool {
	for i := 0; i < len(s); i++ {
		if s[i]|0x20 != name[i] {
			return false
		}
	}
	return true
}

// element is the end of <name ...</name> at i for the first of names that matches: the name followed by a
// \b, the first "</" + the same name (case-insensitive) + \s* + ">" after it, or -1.
func element(s string, i int, names ...string) int {
	for _, name := range names {
		j := i + 1 + len(name)
		if j > len(s) || !asciiFold(s[i+1:j], name) {
			continue
		}
		if r, _ := utf8.DecodeRuneInString(s[j:]); j < len(s) && IsWord(r) {
			continue
		}
		for k := j; ; k++ {
			n := strings.Index(s[k:], "</")
			if n < 0 {
				break
			}
			k += n
			e := k + 2 + len(name)
			if e > len(s) || !asciiFold(s[k+2:e], name) {
				continue
			}
			for e < len(s) {
				r, w := utf8.DecodeRuneInString(s[e:])
				if !IsSpace(r) {
					break
				}
				e += w
			}
			if e < len(s) && s[e] == '>' {
				return e + 1
			}
		}
	}
	return -1
}
