package scaffoldmodule

import "strings"

// The bash tool passed each option value to sed as the replacement of `s|<token>|<value>|g`, so a value
// is sed replacement text, not a literal: `&` and `\0` are the replaced token, and a backslash escapes
// the next character. expandReplacement keeps that reading for every value sed accepts with a plain
// result. Values sed rejected (the bash tool then printed sed errors, left a broken tree and exited 0),
// and the GNU case-conversion and numeric escapes, are reported as unusable instead.

// controlEscapes are the backslash letters GNU sed turns into a control character.
var controlEscapes = map[byte]byte{'a': '\a', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t', 'v': '\v'}

// unsupportedEscapes are the backslash characters with no literal reading: group references 1-9 (sed:
// "invalid reference"), case conversion (L U l u E) and numeric character escapes (c d o x).
const unsupportedEscapes = "123456789LUluEcdox"

// expandReplacement returns what sed writes for replacement text raw when it replaces token; ok is false
// for a value this tool refuses.
func expandReplacement(raw, token string) (text string, ok bool) {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		switch c := raw[i]; c {
		case '|', '\n': // the s command's delimiter, or the end of the command: sed reads the rest as flags
			return "", false
		case '&':
			b.WriteString(token)
		case '\\':
			i++
			if i == len(raw) {
				return "", false
			}
			e := raw[i]
			switch ctl, isCtl := controlEscapes[e]; {
			case e == '0':
				b.WriteString(token)
			case isCtl:
				b.WriteByte(ctl)
			case strings.IndexByte(unsupportedEscapes, e) >= 0:
				return "", false
			default:
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String(), true
}
