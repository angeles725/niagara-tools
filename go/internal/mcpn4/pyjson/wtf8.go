package pyjson

import "unicode/utf8"

// A Python str as this package holds it is UTF-8, or WTF-8 when it holds a lone surrogate. These
// helpers walk it by code point, as str indexing and len() do.

// Chr is chr(r): r as UTF-8, a surrogate as its three WTF-8 bytes.
func Chr(r rune) string { return string(appendWTF8(nil, r)) }

// DecodeRune returns the code point at the start of s and its byte size: a WTF-8 surrogate is
// one code point, and a byte that is neither UTF-8 nor WTF-8 is the surrogate surrogateescape
// decodes it to. An empty s is (utf8.RuneError, 0).
func DecodeRune(s string) (rune, int) {
	r, size := utf8.DecodeRuneInString(s)
	if r == utf8.RuneError && size == 1 {
		if sur, ok := wtf8Surrogate(s); ok {
			return sur, 3
		}
		return 0xdc00 + rune(s[0]), 1
	}
	return r, size
}

// Len is len(s): its code points.
func Len(s string) int {
	n := 0
	for i := 0; i < len(s); n++ {
		_, size := DecodeRune(s[i:])
		i += size
	}
	return n
}
