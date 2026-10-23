package jsonrpc

import "unicode/utf8"

// isPySpace is str.isspace() for one code point: the whitespace set str.strip() removes, wider
// than JSON's (it holds \v, \f, the \x1c-\x1f separators, NEL, NBSP and the Unicode spaces).
func isPySpace(r rune) bool {
	switch {
	case r >= 0x09 && r <= 0x0d, r >= 0x1c && r <= 0x20, r == 0x85, r == 0xa0, r == 0x1680,
		r >= 0x2000 && r <= 0x200a, r == 0x2028, r == 0x2029, r == 0x202f, r == 0x205f, r == 0x3000:
		return true
	}
	return false
}

// Strip is line.strip() on the line Python's stdin decoded (UTF-8, surrogateescape): leading and
// trailing str.isspace() code points go; a byte that is not valid UTF-8 (a lone surrogate to
// Python) is never whitespace.
func Strip(line []byte) []byte {
	for len(line) > 0 {
		r, size := utf8.DecodeRune(line)
		if r == utf8.RuneError && size <= 1 || !isPySpace(r) {
			break
		}
		line = line[size:]
	}
	for len(line) > 0 {
		r, size := utf8.DecodeLastRune(line)
		if r == utf8.RuneError && size <= 1 || !isPySpace(r) {
			break
		}
		line = line[:len(line)-size]
	}
	return line
}
