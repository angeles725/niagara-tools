package dashboardsave

import (
	"regexp"
	"strings"
)

// method is one top-level method (or constructor) of a file: its name, the 1-based lines of its declaration
// and closing brace, and its body text (between the braces). A method inside another method's body (an
// anonymous class, a local class) belongs to the outer one.
type method struct {
	name       string
	first, end int
	body       string
}

var (
	// callOpen is an identifier followed by "(": a method declaration or a call.
	callOpen = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	// bodyOpen follows a declaration's parameter list: an optional throws clause, then the "{".
	bodyOpen = regexp.MustCompile(`^\s*(throws\s+[A-Za-z0-9_.,<>\s]+)?\{`)
	// anonymous ends the text before a "Name(": the name belongs to "new Name(...) {", an anonymous class.
	anonymous = regexp.MustCompile(`\bnew\s+[A-Za-z0-9_.<>]*$`)
	keywords  = map[string]bool{"if": true, "for": true, "while": true, "switch": true, "catch": true,
		"synchronized": true, "try": true, "return": true, "else": true, "do": true}
)

// methodsOf lists the top-level methods of code (comments and literal contents blanked), in text order.
func methodsOf(code string) []method {
	var out []method
	until := 0
	for _, loc := range callOpen.FindAllStringSubmatchIndex(code, -1) {
		at, name := loc[0], code[loc[2]:loc[3]]
		if at < until || keywords[name] || anonymous.MatchString(tail(code[:at])) {
			continue
		}
		close := matching(code, loc[1]-1, '(', ')')
		if close < 0 {
			continue
		}
		b := bodyOpen.FindStringIndex(code[close+1:])
		if b == nil {
			continue
		}
		open := close + b[1]
		end := matching(code, open, '{', '}')
		if end < 0 {
			continue
		}
		out = append(out, method{name: name, first: lineAt(code, at), end: lineAt(code, end), body: code[open+1 : end]})
		until = end
	}
	return out
}

// tail is the last 120 bytes of s.
func tail(s string) string {
	if len(s) > 120 {
		return s[len(s)-120:]
	}
	return s
}

// lineAt is the 1-based line of offset at in code.
func lineAt(code string, at int) int { return strings.Count(code[:at], "\n") + 1 }

// matching is the offset of the bracket closing the one at open, or -1 when it never closes.
func matching(code string, open int, in, out byte) int {
	for depth, i := 0, open; i < len(code); i++ {
		switch code[i] {
		case in:
			depth++
		case out:
			if depth--; depth == 0 {
				return i
			}
		}
	}
	return -1
}
