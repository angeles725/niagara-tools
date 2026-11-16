package delays

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// guarded looks back to the start of the call's method or lambda body (scopeStart) for a positivity guard on
// expr (`expr > 0`, `>= 1`, `== 0`, or the same on `expr().getMillis()`), comments cut, and reports a PASS
// row at the call when one is found; a guard in another method never counts.
// expr is matched as text (the bash lint spliced it into a regular expression) and as a whole name.
func (r *file) guarded(lnum int, expr string) bool {
	if expr == "" {
		return false
	}
	var pats []*regexp.Regexp
	for j := lnum - 1; j >= r.scopeStart(lnum); j-- {
		ln := r.lines[j-1]
		if !strings.Contains(ln, expr) {
			continue
		}
		ln = javasrc.CutComments(ln)
		if !strings.Contains(ln, expr) {
			continue
		}
		if pats == nil {
			pats = guardPatterns(expr)
		}
		for _, p := range pats {
			if p.MatchString(ln) {
				r.emit(lint.Pass, lnum, "guarded at :"+strconv.Itoa(j))
				return true
			}
		}
	}
	return false
}

// guardPatterns are the guard regexps on the literal text expr, which no name character may precede.
func guardPatterns(expr string) []*regexp.Regexp {
	q := regexp.QuoteMeta(expr)
	if isNameByte(expr[0]) {
		q = `(^|[^A-Za-z0-9_$])` + q
	}
	var out []*regexp.Regexp
	for _, tail := range []string{
		`[[:space:]]*>[[:space:]]*0`,
		`[[:space:]]*>=[[:space:]]*1`,
		`[[:space:]]*==[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>[[:space:]]*0`,
		`\(\)\.getMillis\(\)[[:space:]]*>=[[:space:]]*1`,
		`\(\)\.getMillis\(\)[[:space:]]*==[[:space:]]*0`,
	} {
		out = append(out, re(q+tail))
	}
	return out
}
