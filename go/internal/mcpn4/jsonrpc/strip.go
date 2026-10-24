package jsonrpc

import (
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Strip is line.strip() on the line Python's stdin decoded (UTF-8, surrogateescape): leading and
// trailing str.isspace() code points go; a byte that is not valid UTF-8 (a lone surrogate to
// Python) is never whitespace.
func Strip(line []byte) []byte {
	for len(line) > 0 {
		r, size := utf8.DecodeRune(line)
		if r == utf8.RuneError && size <= 1 || !pyjson.IsSpace(r) {
			break
		}
		line = line[size:]
	}
	for len(line) > 0 {
		r, size := utf8.DecodeLastRune(line)
		if r == utf8.RuneError && size <= 1 || !pyjson.IsSpace(r) {
			break
		}
		line = line[:len(line)-size]
	}
	return line
}
