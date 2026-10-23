package javasrc

import "strings"

// CutComments is the toolbelt awk strip_comment() of one line (lint-delays.sh, and the copies in
// lint-arbitrary-ord.sh and lint-write-path.sh): s without a trailing // comment and the blanks before
// it, and with each inline /* ... */ comment replaced by one space. A /* that does not close on the line
// drops the rest of it with the blanks before it. A // or /* inside a "..." string or a '...' char
// literal (\ escapes honoured) is not a comment, and a quote inside a block comment opens no literal.
// It keeps no state across lines: unlike Strip, a line inside a block comment that opened on an
// earlier line is returned as code.
func CutComments(s string) string {
	var q byte // the quote of the literal being read, 0 outside one
	for i := 0; i < len(s); i++ {
		c := s[i]
		if q != 0 {
			if c == '\\' {
				i++
			} else if c == q {
				q = 0
			}
			continue
		}
		switch {
		case c == '"' || c == '\'':
			q = c
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			e := strings.Index(s[i+2:], "*/")
			if e < 0 {
				return trimRightSpace(s[:i])
			}
			s = s[:i] + " " + s[i+2+e+2:] // the loop steps past the space
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			return trimRightSpace(s[:i])
		}
	}
	return s
}

// trimRightSpace is awk's sub(/[[:space:]]+$/, "", s).
func trimRightSpace(s string) string { return strings.TrimRight(s, " \t\n\v\f\r") }
