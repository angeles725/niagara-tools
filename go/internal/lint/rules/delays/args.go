package delays

import "strings"

func trimSpace(s string) string { return strings.Trim(s, " \t\n\v\f\r") }

// callClosed reports whether s (the text after a call's opening paren) holds its closing paren.
func callClosed(s string) bool {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return true
			}
			depth--
		}
	}
	return false
}

// secondArg is the second top-level argument of s, the text after a call's opening paren.
func secondArg(s string) string {
	depth, past := 0, false
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '(':
			depth++
		case c == ')':
			if depth == 0 {
				return out.String()
			}
			depth--
		case c == ',' && depth == 0:
			if past {
				return out.String()
			}
			past = true
			continue
		}
		if past {
			out.WriteByte(c)
		}
	}
	return out.String()
}

// helperName is the method a call text starts with (the last name of Class.fn( or fn( ), or "".
func helperName(s string) string {
	if m := qualCall.FindString(s); m != "" {
		m = callTail.ReplaceAllString(m, "")
		return m[strings.LastIndexByte(m, '.')+1:]
	}
	if m := bareCall.FindString(s); m != "" {
		return callTail.ReplaceAllString(m, "")
	}
	return ""
}

// firstLiteral is the digits of a Math.max first argument that is a bare integer literal (1, 1L), "" for
// anything else (an expression such as 1 + x is no floor). Retro lint-dashboard-write-save Δ3.
func firstLiteral(arg string) string {
	if t := trimSpace(arg); bareLong.MatchString(t) {
		return digits.FindString(t)
	}
	return ""
}
