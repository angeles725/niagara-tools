package pyjson

import (
	"strings"
	"unicode"
)

// Lower is str.lower() as far as a comparison with ASCII text can tell: U+0130 lowers to "i"
// and U+0307 (Unicode SpecialCasing), every other code point to its simple lowercase; bytes
// that are not UTF-8 (a WTF-8 surrogate) are kept.
func Lower(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := DecodeRune(s[i:])
		switch {
		case r == 0x130:
			b.WriteString("i̇")
		case r >= 0xd800 && r <= 0xdfff:
			b.WriteString(s[i : i+size])
		default:
			b.WriteRune(unicode.ToLower(r))
		}
		i += size
	}
	return b.String()
}
