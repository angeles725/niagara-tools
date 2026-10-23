package extwritable

import (
	"regexp"
	"strings"
)

var (
	// nameAttr is the quoted name= value of an annotation.
	nameAttr = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	// The three writes a do<Action> body makes: setX(, getX().setValue( and .set(x,
	setCall      = regexp.MustCompile(`set([A-Z][A-Za-z0-9_]*)\(`)
	setValueCall = regexp.MustCompile(`get([A-Z][A-Za-z0-9_]*)\(\)\.setValue\(`)
	propSetCall  = regexp.MustCompile(`\.set\(([a-zA-Z_][a-zA-Z0-9_]*),`)
)

// cutComments is the awk's own comment strip: it blanks // and /* */ text, carries block-comment state
// across lines and keeps line numbers, but does not know string literals, so a "//" or "/*" inside one
// is read as a comment.
func cutComments(raw []string) []string {
	out := make([]string, len(raw))
	inComment := false
	var b strings.Builder
	for i, ln := range raw {
		b.Reset()
	scan:
		for j := 0; j < len(ln); {
			switch rest := ln[j:]; {
			case inComment:
				if strings.HasPrefix(rest, "*/") {
					inComment = false
					j += 2
				} else {
					j++
				}
			case strings.HasPrefix(rest, "/*"):
				inComment = true
				j += 2
			case strings.HasPrefix(rest, "//"):
				break scan
			default:
				b.WriteByte(ln[j])
				j++
			}
		}
		out[i] = b.String()
	}
	return out
}

// harvestActions returns every @NiagaraAction name and the do<Action> handler name of each. An annotation
// is joined from the line holding "@NiagaraAction" to the line where its parentheses balance; the rest of
// that line is not read for another annotation.
func harvestActions(lines []string) (actions, handlers map[string]bool) {
	actions, handlers = map[string]bool{}, map[string]bool{}
	harvest := func(buf string) {
		m := nameAttr.FindStringSubmatch(buf)
		if m == nil || m[1] == "" {
			return
		}
		actions[m[1]] = true
		handlers["do"+upperFirst(m[1])] = true
	}
	open, buf, depth := false, "", 0
	for _, ln := range lines {
		if !open {
			if !strings.Contains(ln, "@NiagaraAction") {
				continue
			}
			open, buf, depth = true, ln, 0
			hasParen := strings.Contains(ln, "(")
		first:
			for ci := 0; ci < len(ln); ci++ {
				switch ln[ci] {
				case '(':
					depth++
				case ')':
					depth--
					if depth <= 0 && hasParen {
						harvest(buf)
						open, buf = false, ""
						break first
					}
				}
			}
			if open && depth <= 0 {
				harvest(buf)
				open, buf = false, ""
			}
			continue
		}
		buf += " " + ln
	next:
		for ci := 0; ci < len(ln); ci++ {
			switch ln[ci] {
			case '(':
				depth++
			case ')':
				depth--
				if depth <= 0 {
					harvest(buf)
					open, buf = false, ""
					break next
				}
			}
		}
	}
	return actions, handlers
}

// annotations calls fn with each annotation named marker joined from the line holding it (and a "(") to
// the line where its parentheses balance, with the 1-based line it starts on. Later text on that line is
// not read for another annotation, and one still open at the end of the file is dropped.
func annotations(lines []string, marker string, fn func(buf string, line int)) {
	open, buf, depth, start := false, "", 0, 0
	for i, ln := range lines {
		if !open {
			if !strings.Contains(ln, marker) || !strings.Contains(ln, "(") {
				continue
			}
			open, buf, depth, start = true, ln, 0, i+1
		} else {
			buf += " " + ln
		}
		depth += strings.Count(ln, "(") - strings.Count(ln, ")")
		if depth <= 0 {
			fn(buf, start)
			open, buf = false, ""
		}
	}
}

// scanWrites marks every slot body writes through setX(, getX().setValue( or .set(x,
func scanWrites(body string, exempt map[string]bool) {
	for _, m := range setCall.FindAllStringSubmatch(body, -1) {
		exempt[lowerFirst(m[1])] = true
	}
	for _, m := range setValueCall.FindAllStringSubmatch(body, -1) {
		exempt[lowerFirst(m[1])] = true
	}
	for _, m := range propSetCall.FindAllStringSubmatch(body, -1) {
		exempt[m[1]] = true
	}
}

// upperFirst and lowerFirst are awk's toupper/tolower of the first byte (ASCII only in the C locale).
func upperFirst(s string) string {
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		return string(s[0]-'a'+'A') + s[1:]
	}
	return s
}

func lowerFirst(s string) string {
	if s != "" && s[0] >= 'A' && s[0] <= 'Z' {
		return string(s[0]-'A'+'a') + s[1:]
	}
	return s
}
