package servicetypes

import "strings"

const textBlock = `"""`

// blockBody is the text of the brace block that opens on line from (lines joined by "\n"). A nested scope
// (nestedOpener) becomes "{}", so its returns never count as the method's. Strip keeps literal contents, so
// here every byte inside a "…", '…' or """…""" literal becomes "_" (an escape pair "__"): no brace, arrow,
// keyword or return is read from a literal. A \ escapes the next byte on its line only (a line-ending \ in a
// text block escapes the line break); an unterminated "…"/'…' ends at its line end; a text block carries
// across lines and closes on the first unescaped """ (issue #286).
func blockBody(code []string, from int) string {
	var out strings.Builder
	depth, skip, started, inText := 0, 0, false, false
	for i := from; i < len(code); i++ {
		ln := code[i]
		var quote byte
		for j := 0; j < len(ln); j++ {
			ch := ln[j : j+1]
			switch {
			case inText || quote != 0:
				switch {
				case ln[j] == '\\':
					j++
					ch = "__"
				case inText && strings.HasPrefix(ln[j:], textBlock):
					inText = false
					j += 2
					ch = textBlock
				case !inText && ln[j] == quote:
					quote = 0
				default:
					ch = "_"
				}
			case strings.HasPrefix(ln[j:], textBlock):
				inText = true
				j += 2
				ch = textBlock
			case ln[j] == '"' || ln[j] == '\'':
				quote = ln[j]
			case skip > 0:
				if ln[j] == '{' {
					skip++
				} else if ln[j] == '}' {
					if skip--; skip == 0 {
						out.WriteString("{}")
					}
				}
				continue
			case ln[j] == '{':
				if !started {
					depth, started = 1, true
					continue
				}
				if nestedOpener(out.String()) {
					skip = 1
					continue
				}
				depth++
			case ln[j] == '}':
				if depth--; started && depth == 0 {
					return out.String()
				}
			}
			if started && skip == 0 {
				out.WriteString(ch)
			}
		}
		if started {
			out.WriteByte('\n')
		}
	}
	return out.String()
}

// nestedOpener reports whether the "{" that follows pre (literal contents already blanked) opens a scope whose
// returns are not the method's: a block lambda ("-> {" that is not a switch rule), an anonymous class
// ("new X(…) {") or a local class/interface/enum.
func nestedOpener(pre string) bool {
	pre = strings.TrimRight(pre, space)
	seg := pre[strings.LastIndexAny(pre, ";{}")+1:] // the statement text since the last ; { or }
	switch {
	case strings.HasSuffix(pre, "->"):
		return !switchRule(seg)
	case localType.MatchString(seg):
		return true
	case !strings.HasSuffix(pre, ")"):
		return false
	}
	return anonNew.MatchString(pre[:openParen(pre)])
}

// openParen is the index of the "(" that matches the ")" ending s, or 0 when there is none (the awk's
// substr(pre, 1, k-1) of an unmatched scan is empty, as is s[:0]).
func openParen(s string) int {
	depth := 0
	for k := len(s) - 1; k >= 0; k-- {
		switch s[k] {
		case ')':
			depth++
		case '(':
			if depth--; depth == 0 {
				return k
			}
		}
	}
	return 0
}

// switchRule reports whether seg (the statement text before a "->" that precedes "{") is a switch-rule label:
// "default", or "case …" whose labels hold no other "->" (a lambda on a case line), no assignment "=" (an old
// "case X:" label then "r = () -> {") and balanced parentheses ("case X: run(() -> {"); a guard
// ("case T t when (…)") is a label.
func switchRule(seg string) bool {
	seg = arrowEnd.ReplaceAllString(strings.TrimLeft(seg, space), "")
	if seg == "default" {
		return true
	}
	if !caseLabel.MatchString(seg) || strings.Contains(seg, "->") ||
		strings.Contains(compareOp.ReplaceAllString(seg, ""), "=") {
		return false
	}
	depth := 0
	for k := 0; k < len(seg); k++ {
		switch seg[k] {
		case '(':
			depth++
		case ')':
			if depth--; depth < 0 {
				return false
			}
		}
	}
	return depth == 0
}
