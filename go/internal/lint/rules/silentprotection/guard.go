package silentprotection

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// code is a file's stripped lines with literal contents masked by blanks (javasrc.MaskLiterals, so a column
// is the same column of the stripped line), joined by "\n", and the offset each line starts at: braces,
// parentheses and semicolons in it are code.
type code struct {
	text  string
	start []int // start[i] is the offset of 0-based line i
}

func newCode(lines []string) code {
	blank := javasrc.MaskLiterals(lines)
	c := code{text: strings.Join(blank, "\n"), start: make([]int, len(blank))}
	off := 0
	for i, ln := range blank {
		c.start[i] = off
		off += len(ln) + 1
	}
	return c
}

// governs reports whether an if statement whose if( is on 0-based line j holds column col of line i: the
// statement runs from its "if" to the end of its then branch, and of its else branch when one follows.
// A trip below an if statement that already ended is not on its condition.
func (c code) governs(j, i, col int) bool {
	ln := c.text[c.start[j]:]
	if j+1 < len(c.start) {
		ln = c.text[c.start[j]:c.start[j+1]]
	}
	at := c.start[i] + col
	for _, loc := range ifOpen.FindAllStringIndex(ln, -1) {
		open := c.start[j] + loc[1] - 1
		if open < at && at < c.ifEnd(open) {
			return true
		}
	}
	return false
}

// ifEnd is the offset just past the if statement whose condition opens at the "(" at open.
func (c code) ifEnd(open int) int {
	end := c.stmtEnd(c.match(open, '(', ')') + 1)
	if k := c.skip(end); strings.HasPrefix(c.text[k:], "else") && (k+4 == len(c.text) || !identByte(c.text[k+4])) {
		return c.stmtEnd(k + 4)
	}
	return end
}

// stmtEnd is the offset just past the statement starting at or after k: a block, a nested if (with its
// else), or the text up to a ";" outside parentheses (a braced block met on the way ends it).
func (c code) stmtEnd(k int) int {
	k = c.skip(k)
	if k >= len(c.text) {
		return len(c.text)
	}
	if c.text[k] == '{' {
		return c.match(k, '{', '}') + 1
	}
	if loc := ifOpen.FindStringIndex(c.text[k:]); loc != nil && strings.TrimLeft(c.text[k:k+loc[0]], space) == "" {
		return c.ifEnd(k + loc[1] - 1)
	}
	depth := 0
	for ; k < len(c.text); k++ {
		switch c.text[k] {
		case '(':
			depth++
		case ')':
			depth--
		case '{':
			if depth == 0 {
				return c.match(k, '{', '}') + 1
			}
		case ';':
			if depth <= 0 {
				return k + 1
			}
		}
	}
	return len(c.text)
}

// match is the offset of the byte closing the one at open (len(text) when it never closes).
func (c code) match(open int, l, r byte) int {
	depth := 0
	for k := open; k < len(c.text); k++ {
		switch c.text[k] {
		case l:
			depth++
		case r:
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return len(c.text)
}

// skip is the first offset at or after k that is not a blank.
func (c code) skip(k int) int {
	for k < len(c.text) && strings.IndexByte(space, c.text[k]) >= 0 {
		k++
	}
	return k
}

func identByte(b byte) bool {
	return b == '_' || b == '$' || 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '0' <= b && b <= '9'
}
