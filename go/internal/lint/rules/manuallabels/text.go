package manuallabels

import (
	"strings"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
)

func firstRune(s string) rune { r, _ := utf8.DecodeRuneInString(s); return r }

// label is one quoted label of the manual and its 1-based line.
type label struct {
	text string
	line int
}

// closers are the label quotes: "…", “…” and «…».
var closers = map[rune]rune{'"': '"', '“': '”', '«': '»'}

// labels are the quoted labels of each line, as re.finditer('"([^"\n]{2,60})"|“([^”\n]{2,60})”|«([^»\n]{2,60})»')
// finds them (an opening quote whose content is too short or too long is skipped, and the scan goes on at
// the next code point), stripped, keeping those with a letter ([^\W\d_]) and no "://".
func labels(text string) []label {
	var out []label
	for n, line := range strings.Split(text, "\n") {
		for i := 0; i < len(line); {
			r, w := utf8.DecodeRuneInString(line[i:])
			if c, ok := closers[r]; ok {
				if j := strings.IndexRune(line[i+w:], c); j >= 0 {
					if k := utf8.RuneCountInString(line[i+w : i+w+j]); k >= 2 && k <= 60 {
						if l := htmltext.Strip(line[i+w : i+w+j]); hasLetter(l) && !strings.Contains(l, "://") {
							out = append(out, label{l, n + 1})
						}
						i += w + j + utf8.RuneLen(c)
						continue
					}
				}
			}
			i += w
		}
	}
	return out
}

func hasLetter(s string) bool {
	for _, r := range s {
		if htmltext.IsAlnum(r) && !htmltext.IsDecimal(r) {
			return true
		}
	}
	return false
}

// units are the UI text units of one decoded file (\uNNNN escapes, then HTML entities, decoded): HTML text
// nodes (>…<), string literals ("…", '…' with backslash escapes on one line, `…`) and lexicon or properties
// values (key=value), each as the three Python regexes find them; empty units are dropped.
func units(s string) []string {
	s = htmltext.Unescape(unicodeEscapes(s))
	var out []string
	add := func(u string) {
		if u != "" {
			out = append(out, u)
		}
	}
	textNodes(s, add)
	literals(s, add)
	values(s, add)
	return out
}

// unicodeEscapes is re.sub(r"\\u([0-9a-fA-F]{4})", chr): a surrogate code point is written as its three
// WTF-8 bytes, so it stays one unmatched code point as in Python.
func unicodeEscapes(s string) string {
	i := strings.Index(s, `\u`)
	if i < 0 {
		return s
	}
	var b strings.Builder
	for ; i >= 0; i = strings.Index(s, `\u`) {
		v, ok := hex4(s[i+2:])
		if !ok {
			b.WriteString(s[:i+1])
			s = s[i+1:]
			continue
		}
		b.WriteString(s[:i])
		if v >= 0xD800 && v <= 0xDFFF {
			b.Write([]byte{0xE0 | byte(v>>12), 0x80 | byte(v>>6&0x3F), 0x80 | byte(v&0x3F)})
		} else {
			b.WriteRune(v)
		}
		s = s[i+6:]
	}
	return b.String() + s
}

func hex4(s string) (rune, bool) {
	if len(s) < 4 {
		return 0, false
	}
	var v rune
	for _, c := range []byte(s[:4]) {
		switch {
		case c >= '0' && c <= '9':
			v = v<<4 | rune(c-'0')
		case c|0x20 >= 'a' && c|0x20 <= 'f':
			v = v<<4 | rune(c|0x20-'a'+10)
		default:
			return 0, false
		}
	}
	return v, true
}

// textNodes is re.finditer(r">([^<>]+)<").
func textNodes(s string, add func(string)) {
	for i := strings.IndexByte(s, '>'); i >= 0; {
		k := strings.IndexAny(s[i+1:], "<>")
		if k < 0 {
			return
		}
		next := i + 1 + k
		if s[next] == '<' && k > 0 {
			add(s[i+1 : next])
			next++
		}
		j := strings.IndexByte(s[next:], '>')
		if j < 0 {
			return
		}
		i = next + j
	}
}

// literals is re.finditer(r'"((?:[^"\\\n]|\\.)*)"|\'((?:[^\'\\\n]|\\.)*)\'|`([^`]*)`'): an opening quote
// that is never closed (a newline, or the end) matches nothing and the scan goes on at the next byte.
func literals(s string, add func(string)) {
	for i := 0; i < len(s); {
		k := strings.IndexAny(s[i:], "\"'`")
		if k < 0 {
			return
		}
		i += k
		if end := closeQuote(s, i); end > 0 {
			add(s[i+1 : end-1])
			i = end
			continue
		}
		i++
	}
}

// closeQuote is the end of the literal opened at i (after its closing quote), or -1.
func closeQuote(s string, i int) int {
	q := s[i]
	if q == '`' {
		if j := strings.IndexByte(s[i+1:], '`'); j >= 0 {
			return i + j + 2
		}
		return -1
	}
	for j := i + 1; j < len(s); j++ {
		switch s[j] {
		case q:
			return j + 1
		case '\n':
			return -1
		case '\\':
			if j+1 >= len(s) || s[j+1] == '\n' {
				return -1
			}
			_, w := utf8.DecodeRuneInString(s[j+1:])
			j += w
		}
	}
	return -1
}

// values is re.finditer(r"(?m)^\s*[^#!=\s][^=\n]*=(.*)$"): from a line start, whitespace (newlines included)
// is skipped; the first other code point must not be '#', '!' or '='; the value runs from the first '=' of
// that line to its end. Every line start the skip crossed reaches the same value, so the scan goes on after
// that line.
func values(s string, add func(string)) {
	for p := 0; p < len(s); {
		q := p
		for q < len(s) {
			r, w := utf8.DecodeRuneInString(s[q:])
			if !htmltext.IsSpace(r) {
				break
			}
			q += w
		}
		end := len(s)
		if k := strings.IndexByte(s[q:], '\n'); k >= 0 {
			end = q + k
		}
		if q < end && s[q] != '#' && s[q] != '!' && s[q] != '=' {
			if eq := strings.IndexByte(s[q:end], '='); eq >= 0 {
				add(s[q+eq+1 : end])
			}
		}
		p = end + 1
	}
}
