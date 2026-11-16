// Package rowpath spells a path for a report row. A path comes from a repository or an archive, so it may
// hold any byte; a row is one line. check-installed-line and lint-ntpl-manifest-deps print their paths
// through Display.
package rowpath

import (
	"fmt"
	"strings"
)

// cEscapes are the control bytes git writes as a letter escape when it quotes a path.
var cEscapes = map[byte]string{'\a': `\a`, '\b': `\b`, '\t': `\t`, '\n': `\n`, '\v': `\v`, '\f': `\f`, '\r': `\r`, '"': `\"`, '\\': `\\`}

// Display is a path as a row prints it: as it is on disk, unless it holds a control byte, a double quote or
// a backslash; then it is in double quotes with those bytes C-escaped (the other control bytes as three
// octal digits), which is how git prints such a path with core.quotePath off. A row is therefore always one
// line, and a non-ASCII path is printed unchanged.
func Display(path string) string {
	if !needsQuote(path) {
		return path
	}
	var b strings.Builder
	b.WriteByte('"')
	for i := 0; i < len(path); i++ {
		switch c := path[i]; {
		case cEscapes[c] != "":
			b.WriteString(cEscapes[c])
		case isControl(c):
			fmt.Fprintf(&b, `\%03o`, c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func needsQuote(path string) bool {
	for i := 0; i < len(path); i++ {
		if c := path[i]; isControl(c) || c == '"' || c == '\\' {
			return true
		}
	}
	return false
}

func isControl(c byte) bool { return c < 0x20 || c == 0x7f }
