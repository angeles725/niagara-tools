// Package htmltext is the Python text layer of the kit's HTML-text lints, byte for byte as CPython 3.15
// produces it: a file as open(path, encoding="utf-8", errors="replace").read() returns it (Decode),
// html.unescape (Unescape), an HTML page's text with its markup dropped and its line numbers kept
// (DropMarkup), and the str facts and folds those need (unicode.go). Its first consumer is
// lint-manual-labels (rules/manuallabels).
package htmltext

import (
	"strings"
	"unicode/utf8"
)

// Decode is a file's bytes as Python reads them in text mode: UTF-8 with each maximal invalid subsequence
// replaced by one U+FFFD, then universal newlines ("\r\n" and a lone "\r" become "\n").
func Decode(b []byte) string {
	s := string(b)
	if !utf8.ValidString(s) {
		var sb strings.Builder
		for i := 0; i < len(s); {
			if r, n := utf8.DecodeRuneInString(s[i:]); r != utf8.RuneError || n > 1 {
				sb.WriteString(s[i : i+n])
				i += n
				continue
			}
			sb.WriteRune(utf8.RuneError)
			i = invalidEnd(s, i)
		}
		s = sb.String()
	}
	if strings.IndexByte(s, '\r') >= 0 {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	}
	return s
}

// invalidEnd is the end of the invalid sequence starting at i: CPython replaces the start byte and the
// continuation bytes that were still valid for it, up to the first byte that breaks the sequence.
func invalidEnd(s string, i int) int {
	c, need := s[i], 0
	lo, hi := byte(0x80), byte(0xBF) // the allowed range of the first continuation byte
	switch {
	case c >= 0xC2 && c <= 0xDF:
		need = 1
	case c >= 0xE0 && c <= 0xEF:
		need = 2
		if c == 0xE0 {
			lo = 0xA0
		} else if c == 0xED {
			hi = 0x9F
		}
	case c >= 0xF0 && c <= 0xF4:
		need = 3
		if c == 0xF0 {
			lo = 0x90
		} else if c == 0xF4 {
			hi = 0x8F
		}
	}
	j := i + 1
	for k := 0; k < need && j < len(s); k, j = k+1, j+1 {
		if b := s[j]; (k == 0 && (b < lo || b > hi)) || (k > 0 && (b < 0x80 || b > 0xBF)) {
			break
		}
	}
	return j
}
