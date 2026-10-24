package pyjson

import (
	"fmt"
	"unicode/utf8"
)

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

// EncodeUTF8 checks s.encode("utf-8"): a lone surrogate is the UnicodeEncodeError (a ValueError)
// naming its position, or the run of them, in s.
func EncodeUTF8(s string) error {
	start, end, first := -1, -1, rune(0)
	for i, n := 0, 0; i < len(s) && (start < 0 || end == n); n++ {
		r, size := DecodeRune(s[i:])
		i += size
		switch surrogate := r >= 0xd800 && r <= 0xdfff; {
		case surrogate && start < 0:
			start, end, first = n, n+1, r
		case surrogate:
			end = n + 1
		}
	}
	switch {
	case start < 0:
		return nil
	case end == start+1:
		return &ValueError{Msg: fmt.Sprintf("'utf-8' codec can't encode character '\\u%04x' in position %d: "+
			"surrogates not allowed", first, start)}
	}
	return &ValueError{Msg: fmt.Sprintf("'utf-8' codec can't encode characters in position %d-%d: "+
		"surrogates not allowed", start, end-1)}
}
