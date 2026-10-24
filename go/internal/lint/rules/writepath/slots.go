package writepath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

var (
	propName    = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	coveredName = regexp.MustCompile(`name[[:space:]]*=[[:space:]]*"([A-Za-z][A-Za-z0-9_]*)"`)
)

// names is the covered set of the STALE pass: every name = "X" in the code of any Java file under the
// matrix root (an @NiagaraProperty of any flag, an @NiagaraAction, or any other name = "..." on a line);
// a name only a comment mentions covers no row (truth fix T3; the bash grep read raw lines).
func (s *scan) names(files []string) (map[string]bool, error) {
	set := map[string]bool{}
	for _, p := range files {
		f, err := s.file(p)
		if err != nil {
			return nil, err
		}
		for _, ln := range f.Stripped() {
			for _, m := range coveredName.FindAllStringSubmatch(ln, -1) {
				set[m[1]] = true
			}
		}
	}
	return set, nil
}

// operatorSlots are the names of the @NiagaraProperty annotations over files that carry OPERATOR or "o",
// sorted: an annotation runs from a line holding "@NiagaraProperty" until its parentheses balance, its
// name is the first name = "..." in it. It is read in code (truth fix T3): a commented-out annotation or
// an OPERATOR in a comment is not a write path, a parenthesis inside a literal does not keep the
// annotation open, and an annotation left open at a file's end ends there (the bash awk read raw lines,
// one `xargs awk` run over every file, so it went on into the next file).
func (s *scan) operatorSlots(files []string) ([]string, error) {
	set := map[string]bool{}
	for _, p := range files {
		f, err := s.file(p)
		if err != nil {
			return nil, err
		}
		text := f.Stripped()     // comments removed, literal text kept (names, "o")
		code := linescan.Code(f) // literal contents removed too (markers, parentheses)
		var in, named, op bool
		var buf, name string // buf: the annotation's lines joined by " ", kept until its first name = "..."
		depth := 0
		for i, ln := range text {
			switch {
			case in:
				if !named {
					buf += " " + ln
				}
			case strings.Contains(code[i], "@NiagaraProperty"):
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
			op = op || strings.Contains(code[i], "OPERATOR") || strings.Contains(ln, `"o"`)
			if depth += strings.Count(code[i], "(") - strings.Count(code[i], ")"); depth <= 0 {
				if name != "" && op {
					set[name] = true
				}
				in = false
			}
		}
	}
	return sorted(set), nil
}
