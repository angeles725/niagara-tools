package triageconsole

import "strings"

// subsep is awk's SUBSEP, the separator inside a grouping key; the keys are spelled exactly as the awk built
// them because gawk's for-in order (awkorder) hashes their bytes.
const subsep = "\x1c"

// from is awk substr(s, m) for m >= 1: s from its m-th byte on ("" past the end).
func from(s string, m int) string {
	if m-1 >= len(s) {
		return ""
	}
	return s[m-1:]
}

// substr is awk substr(s, m, n) for m >= 1: at most n bytes from the m-th ("" for n <= 0).
func substr(s string, m, n int) string {
	s = from(s, m)
	if n <= 0 {
		return ""
	}
	if n < len(s) {
		return s[:n]
	}
	return s
}

// split is awk split(s, a, sep) for a one-character literal separator: no element for an empty s.
func split(s, sep string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, sep)
}

// set is the array split() fills, read with `in`.
func set(list string) map[string]bool {
	m := map[string]bool{}
	for _, e := range split(list, ",") {
		m[e] = true
	}
	return m
}

// norm is the awk norm(): every run of digits becomes N.
func norm(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if isDigit(s[i]) {
			b.WriteByte('N')
			for i+1 < len(s) && isDigit(s[i+1]) {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// lower is awk tolower() under LC_ALL=C: ASCII letters only.
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// fields123 is awk's $1, $2 and $3 under the default FS: blank-separated (space, tab, newline), leading blanks
// skipped.
func fields123(line string) (f1, f2, f3 string) {
	f := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '\t' || r == '\n' })
	if len(f) > 0 {
		f1 = f[0]
	}
	if len(f) > 1 {
		f2 = f[1]
	}
	if len(f) > 2 {
		f3 = f[2]
	}
	return f1, f2, f3
}

// escaper processes `awk -v name=value` values as gawk 5 does: C escapes, up to 3 octal digits, up to 2 hex
// digits after \x, a dropped trailing backslash and backslash-newline, and any other \c read as c with one
// warning per character for the whole run (a \x without hex digits warns every time).
type escaper struct {
	warned [256]bool
	warn   func(string)
}

func (e *escaper) value(v string) string {
	var b strings.Builder
	for i := 0; i < len(v); i++ {
		if v[i] != '\\' {
			b.WriteByte(v[i])
			continue
		}
		i++
		if i == len(v) {
			break
		}
		c := v[i]
		switch {
		case strings.IndexByte("abfnrtv", c) >= 0:
			b.WriteByte("\a\b\f\n\r\t\v"[strings.IndexByte("abfnrtv", c)])
		case c == '\n':
		case c == '"' || c == '\\':
			b.WriteByte(c)
		case c >= '0' && c <= '7':
			n := int(c - '0')
			for k := 0; k < 2 && i+1 < len(v) && v[i+1] >= '0' && v[i+1] <= '7'; k++ {
				i++
				n = n*8 + int(v[i]-'0')
			}
			b.WriteByte(byte(n))
		case c == 'x':
			n, digits := 0, 0
			for digits < 2 && i+1 < len(v) && hexVal(v[i+1]) >= 0 {
				i++
				n = n*16 + hexVal(v[i])
				digits++
			}
			if digits == 0 {
				e.warn("awk: warning: no hex digits in `\\x' escape sequence\n")
				b.WriteByte('x')
				continue
			}
			b.WriteByte(byte(n))
		default:
			if !e.warned[c] {
				e.warned[c] = true
				ch := string([]byte{c}) // the raw byte, as gawk's %c prints it (not a rune)
				e.warn("awk: warning: escape sequence `\\" + ch + "' treated as plain `" + ch + "'\n")
			}
			b.WriteByte(c)
		}
	}
	return b.String()
}

func hexVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}
