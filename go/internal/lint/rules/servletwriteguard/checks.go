package servletwriteguard

import (
	"regexp"
	"strings"
)

// check is one target permission check: the variables it names, and where on its line its expression starts.
type check struct {
	names map[string]bool
	at    int
}

// guards reports whether c guards a write whose receiver names the variables w: a shared variable, or either
// side naming none (a receiver the lint cannot read, such as getTarget().set(...), matches any).
func (c check) guards(w map[string]bool) bool {
	if len(c.names) == 0 || len(w) == 0 {
		return true
	}
	for n := range w {
		if c.names[n] {
			return true
		}
	}
	return false
}

// receiverStart is where the receiver expression ending at byte end of s starts: back over identifiers, dots
// and balanced parenthesised groups (a cast, a call), to the first other character outside a group.
func receiverStart(s string, end int) int {
	depth := 0
	for i := end; i > 0; i-- {
		c := s[i-1]
		switch {
		case c == ')':
			depth++
		case c == '(' && depth == 0:
			return i
		case c == '(':
			depth--
		case depth > 0 || isIdentByte(c) || c == '.':
		default:
			return i
		}
	}
	return 0
}

// argEnd is the byte index of the ')' that closes the call whose '(' is at open, or len(s).
func argEnd(s string, open int) int {
	depth := 0
	for i := open; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return len(s)
}

var firstArg = regexp.MustCompile(`^[^,)]*`)

// checksOn are the target checks on line ln, each with the variables it names; requireTargetWrite counts only
// when helperOK. prev is the last code character on an earlier line of the method (0 when none): a check is
// counted only when its result is used, not as a bare statement whose value is discarded.
func checksOn(ln string, prev byte, helperOK bool) []check {
	var out []check
	for _, loc := range targetCall.FindAllStringIndex(ln, -1) {
		tok := ln[loc[0]:loc[1]]
		open := loc[0] + strings.IndexByte(tok, '(')
		var names map[string]bool
		start := loc[0]
		switch {
		case strings.Contains(tok, "requireTargetWrite"):
			if !helperOK {
				continue
			}
			start = open - len("requireTargetWrite")
			names = varNames(firstArg.FindString(ln[open+1:]))
		case strings.HasPrefix(tok, "getPermissionsFor"):
			if loc[0] > 0 && ln[loc[0]-1] == '.' {
				start = receiverStart(ln, loc[0]-1)
			}
			names = varNames(ln[open+1 : argEnd(ln, open)])
		default: // .getPermissions(cx) / .canWrite(): the receiver is what is checked
			start = receiverStart(ln, loc[0])
			names = varNames(ln[start:loc[0]])
		}
		if !used(ln[:start], prev) {
			continue
		}
		out = append(out, check{names: names, at: loc[0]})
	}
	return out
}

// used reports whether an expression starting after text before (on its line; prev is the last code character
// of the earlier lines) is used rather than a statement of its own: a statement starts at the line start after
// ';', '{' or '}', or after one of them on the same line.
func used(before string, prev byte) bool {
	b := strings.TrimRight(before, " \t")
	if b == "" {
		return prev != 0 && !strings.ContainsRune(";{}", rune(prev))
	}
	return !strings.ContainsRune(";{}", rune(b[len(b)-1]))
}

// lastCode is the last non-space character of s, or 0.
func lastCode(s string) byte {
	t := strings.TrimRight(s, " \t\r")
	if t == "" {
		return 0
	}
	return t[len(t)-1]
}
