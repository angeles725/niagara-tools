package arbitraryord

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// The in-method proofs that a BOrd.make(<v>) is not an arbitrary ord (kit #459). They read the file's
// stripped lines (comments removed, literal text kept) one statement per line, inside the javasrc scope
// that holds the call; anything they cannot follow leaves the row in place.

var (
	ordArg   = regexp.MustCompile(`BOrd\.make\([[:space:]]*([a-z][A-Za-z0-9_]*)[[:space:]]*\)`)
	ident    = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
	constant = regexp.MustCompile(`^[[:space:]]*(([A-Za-z_][A-Za-z0-9_]*\.)*[A-Z][A-Z0-9_]*|"[^"]*")[[:space:]]*\+`)
)

// maxDepth bounds how far a value is followed through assignments.
const maxDepth = 8

// tracer follows one call's ord variable inside its scope.
type tracer struct {
	text  []string // javasrc stripped lines
	scope javasrc.Method
	line  int // 0-based line of the call
}

// newTracer is the tracer of the BOrd.make(<v>) on 0-based line i and its variable v, ok false when the
// call is outside every scope or its argument is not a plain variable.
func newTracer(f *javasrc.File, i int) (t *tracer, v string, ok bool) {
	m := ordArg.FindStringSubmatch(f.Stripped()[i])
	scope, in := javasrc.Enclosing(f.Scopes(), i+1)
	if m == nil || !in {
		return nil, "", false
	}
	return &tracer{text: f.Stripped(), scope: scope, line: i}, m[1], true
}

// assigned is the right-hand side of v's last assignment (or declaration) before the call in the scope.
func (t *tracer) assigned(v string) (string, bool) {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_.])` + regexp.QuoteMeta(v) + `[[:space:]]*=([^=][^;]*);`)
	for i := t.line - 1; i >= t.scope.Start-1 && i >= 0; i-- {
		if m := re.FindStringSubmatch(t.text[i]); m != nil {
			return m[2], true
		}
	}
	return "", false
}

// traversalGuarded reports whether an `if` before the call in the scope tests v.contains("..").
func (t *tracer) traversalGuarded(v string) bool {
	re := regexp.MustCompile(`\bif[[:space:]]*\(.*\b` + regexp.QuoteMeta(v) + `[[:space:]]*\.[[:space:]]*contains[[:space:]]*\([[:space:]]*"\.\."[[:space:]]*\)`)
	for i := t.scope.Start - 1; i < t.line && i >= 0; i++ {
		if re.MatchString(t.text[i]) {
			return true
		}
	}
	return false
}

// prefixedAndGuarded is #459 (a): v is assigned a constant (an UPPER_CASE name or a string literal) plus
// the rest, so the ord cannot leave that prefix, and every variable in the rest is traversal-guarded, or
// assigned only from guarded variables, before the call.
func (t *tracer) prefixedAndGuarded(v string) bool {
	rhs, ok := t.assigned(v)
	return ok && constant.MatchString(rhs) && t.guardedVars(rhs[constant.FindStringIndex(rhs)[1]:], 0)
}

// guardedVars reports whether every local variable named in expr is guarded (see prefixedAndGuarded).
func (t *tracer) guardedVars(expr string, depth int) bool {
	for _, v := range variables(expr) {
		if t.traversalGuarded(v) {
			continue
		}
		rhs, ok := t.assigned(v)
		if !ok || depth == maxDepth || !t.guardedVars(rhs, depth+1) {
			return false
		}
	}
	return true
}

// variables are the lowercase identifiers of expr that name values: not a method name (before "("), not a
// member (after "."), not a literal's text, not a Java keyword or literal.
func variables(expr string) []string {
	expr = javasrc.BlankLine(expr)
	var out []string
	for _, loc := range ident.FindAllStringIndex(expr, -1) {
		name, before, after := expr[loc[0]:loc[1]], strings.TrimRight(expr[:loc[0]], " \t"), strings.TrimLeft(expr[loc[1]:], " \t")
		if name[0] < 'a' || name[0] > 'z' || strings.HasSuffix(before, ".") || strings.HasPrefix(after, "(") || keywords[name] {
			continue
		}
		out = append(out, name)
	}
	return out
}

var keywords = map[string]bool{"new": true, "null": true, "true": true, "false": true, "this": true, "instanceof": true}
