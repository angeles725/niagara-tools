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

// harvestActions returns every @NiagaraAction name and the do<Action> handler name of each. An annotation
// is joined from the line holding "@NiagaraAction" to the line where its parentheses balance; the rest of
// that line is not read for another annotation. The marker and the parentheses are read in code, the
// lines with literal contents blanked (see javasrc.BlankLiterals), and the name in lines.
func harvestActions(lines, code []string) (actions, handlers map[string]bool) {
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
	for i, ln := range code {
		if !open {
			if !strings.Contains(ln, "@NiagaraAction") {
				continue
			}
			open, buf, depth = true, lines[i], 0
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
		buf += " " + lines[i]
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
// not read for another annotation, and one still open at the end of the file is dropped. The marker and
// the parentheses are read in code (literal contents blanked), and buf joins lines.
func annotations(lines, code []string, marker string, fn func(buf string, line int)) {
	open, buf, depth, start := false, "", 0, 0
	for i, ln := range code {
		if !open {
			if !strings.Contains(ln, marker) || !strings.Contains(ln, "(") {
				continue
			}
			open, buf, depth, start = true, lines[i], 0, i+1
		} else {
			buf += " " + lines[i]
		}
		depth += strings.Count(ln, "(") - strings.Count(ln, ")")
		if depth <= 0 {
			fn(buf, start)
			open, buf = false, ""
		}
	}
}

// scanWrites marks every slot body writes through setX(, getX().setValue( or .set(x, where setX and getX
// are whole names (resetX( or forgetX() write nothing).
func scanWrites(body string, exempt map[string]bool) {
	for _, re := range []*regexp.Regexp{setCall, setValueCall} {
		for _, m := range re.FindAllStringSubmatchIndex(body, -1) {
			if m[0] == 0 || !identByte(body[m[0]-1]) {
				exempt[lowerFirst(body[m[2]:m[3]])] = true
			}
		}
	}
	for _, m := range propSetCall.FindAllStringSubmatch(body, -1) {
		exempt[m[1]] = true
	}
}

// identByte reports whether c can continue a Java identifier (ASCII letters, digits, _ and $).
func identByte(c byte) bool {
	return c == '_' || c == '$' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
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
