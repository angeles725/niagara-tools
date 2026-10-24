package writepath

import (
	"regexp"
	"strings"
)

var (
	propName    = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	coveredName = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([A-Za-z][A-Za-z0-9_]*)"`)
)

// names is the covered set of the STALE pass: every name = "X" in any Java file under the matrix root
// (an @NiagaraProperty of any flag, an @NiagaraAction, or any other name = "..." on a line).
func (s *scan) names(files []string) (map[string]bool, error) {
	set := map[string]bool{}
	for _, p := range files {
		lines, err := s.lines(p)
		if err != nil {
			return nil, err
		}
		for _, ln := range lines {
			for _, m := range coveredName.FindAllStringSubmatch(ln, -1) {
				set[m[1]] = true
			}
		}
	}
	return set, nil
}

// operatorSlots are the names of the @NiagaraProperty annotations over files that carry OPERATOR or "o",
// sorted: an annotation runs from a line holding "@NiagaraProperty" until its parentheses balance, its
// name is the first name = "..." in it. One scan runs over every file in turn, as one `xargs awk` did,
// so an annotation left open at a file's end goes on into the next file.
func (s *scan) operatorSlots(files []string) ([]string, error) {
	set := map[string]bool{}
	var in, named, op bool
	var buf, name string // buf: the annotation's lines joined by " ", kept until its first name = "..."
	depth := 0
	for _, p := range files {
		lines, err := s.lines(p)
		if err != nil {
			return nil, err
		}
		for _, ln := range lines {
			switch {
			case in:
				if !named {
					buf += " " + ln
				}
			case strings.Contains(ln, "@NiagaraProperty"):
				in, named, op, buf, name, depth = true, false, false, ln, "", 0
			default:
				continue
			}
			if !named {
				if m := propName.FindStringSubmatch(buf); m != nil {
					named, name, buf = true, m[1], ""
				}
			}
			// the join adds a blank, so OPERATOR and "o" never span two lines
			op = op || strings.Contains(ln, "OPERATOR") || strings.Contains(ln, `"o"`)
			if depth += strings.Count(ln, "(") - strings.Count(ln, ")"); depth <= 0 {
				if name != "" && op {
					set[name] = true
				}
				in = false
			}
		}
	}
	return sorted(set), nil
}
