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

// tracer follows values inside the scope holding one statement.
type tracer struct {
	f     *javasrc.File
	text  []string // javasrc stripped lines
	scope javasrc.Method
	line  int // 0-based line of the statement
}

// at is the tracer of the statement on 0-based line i, ok false outside every scope.
func at(f *javasrc.File, i int) (*tracer, bool) {
	scope, in := javasrc.Enclosing(f.Scopes(), i+1)
	return &tracer{f: f, text: f.Stripped(), scope: scope, line: i}, in
}

// newTracer is the tracer of the BOrd.make(<v>) on 0-based line i and its variable v, ok false when the
// call is outside every scope or its argument is not a plain variable.
func newTracer(f *javasrc.File, i int) (t *tracer, v string, ok bool) {
	m := ordArg.FindStringSubmatch(f.Stripped()[i])
	t, in := at(f, i)
	if m == nil || !in {
		return nil, "", false
	}
	return t, m[1], true
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

var (
	alarmSource = regexp.MustCompile(`^([a-z][A-Za-z0-9_]*)[[:space:]]*\.[[:space:]]*getSource[[:space:]]*\([[:space:]]*\)([[:space:]]*\.[[:space:]]*toString[[:space:]]*\([[:space:]]*\))?$`)
	callExpr    = regexp.MustCompile(`^([a-z][A-Za-z0-9_]*)[[:space:]]*\((.*)\)$`)
	returns     = regexp.MustCompile(`\breturn[[:space:]]+([^;]+);`)
)

// fromAlarmSource is #459 (b): every origin of v is a BAlarmRecord's getSource(), a station value. v is
// followed through its assignment in the scope or, for a parameter of a private method, the argument every
// call of that method in the file passes; an expression through a private method's every return.
func (t *tracer) fromAlarmSource(v string, depth int) bool {
	if depth == maxDepth {
		return false
	}
	if rhs, ok := t.assigned(v); ok {
		return t.origin(strings.TrimSpace(rhs), depth+1)
	}
	sig, ok := t.signature()
	if !ok || !strings.Contains(sig.head, "private") {
		return false
	}
	pos := -1
	for k, p := range sig.params {
		if p.name == v {
			pos = k
		}
	}
	calls := t.callsOf(t.scope.Name)
	if pos < 0 || len(calls) == 0 {
		return false
	}
	for _, c := range calls {
		if pos >= len(c.args) || !c.at.origin(c.args[pos], depth+1) {
			return false
		}
	}
	return true
}

// origin reports whether expr, at t's statement, is a BAlarmRecord's getSource() (see fromAlarmSource).
func (t *tracer) origin(expr string, depth int) bool {
	expr = strings.TrimSpace(expr)
	if m := alarmSource.FindStringSubmatch(expr); m != nil {
		return t.declaredAs(m[1], "BAlarmRecord")
	}
	if ident.FindString(expr) == expr && expr != "" && expr[0] >= 'a' && expr[0] <= 'z' {
		return t.fromAlarmSource(expr, depth)
	}
	m := callExpr.FindStringSubmatch(expr)
	if m == nil || depth == maxDepth {
		return false
	}
	var rets []*tracer
	var exprs []string
	for _, sc := range t.f.Scopes() {
		if sc.Name != m[1] {
			continue
		}
		u := &tracer{f: t.f, text: t.text, scope: sc, line: sc.Start - 1}
		if sig, ok := u.signature(); !ok || !strings.Contains(sig.head, "private") {
			return false
		}
		for i := sc.Start - 1; i < sc.End && i < len(t.text); i++ {
			for _, r := range returns.FindAllStringSubmatch(t.text[i], -1) {
				rets, exprs = append(rets, &tracer{f: t.f, text: t.text, scope: sc, line: i}), append(exprs, r[1])
			}
		}
	}
	if len(exprs) == 0 {
		return false
	}
	for k, e := range exprs {
		// a catch's fallback literal ("" or null) resolves no ord of its own
		if e = strings.TrimSpace(e); e != `""` && e != "null" && !rets[k].origin(e, depth+1) {
			return false
		}
	}
	return true
}

// declaredAs reports whether name is declared with type typ in t's scope signature or a statement before t.
func (t *tracer) declaredAs(name, typ string) bool {
	if sig, ok := t.signature(); ok {
		for _, p := range sig.params {
			if p.name == name && p.typ == typ {
				return true
			}
		}
	}
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(typ) + `[[:space:]]+` + regexp.QuoteMeta(name) + `\b`)
	for i := t.scope.Start - 1; i <= t.line && i >= 0 && i < len(t.text); i++ {
		if re.MatchString(t.text[i]) {
			return true
		}
	}
	return false
}

type param struct{ typ, name string }

type signature struct {
	head   string // the declaration before its parameter list ("private static String safeSource")
	params []param
}

// signature is the declaration of t's scope: the lines up to its opening brace, read back from it.
func (t *tracer) signature() (signature, bool) {
	start := t.scope.Start - 1
	if start < 0 || start >= len(t.text) || t.scope.Name == "" {
		return signature{}, false
	}
	re := regexp.MustCompile(`^(.*\b` + regexp.QuoteMeta(t.scope.Name) + `)[[:space:]]*\(([^)]*)\)`)
	joined := ""
	for i := start; i >= 0 && i > start-5; i-- {
		joined = t.text[i] + " " + joined
		if m := re.FindStringSubmatch(strings.TrimSpace(joined)); m != nil {
			sig := signature{head: m[1]}
			for _, p := range splitArgs(m[2]) {
				f := strings.Fields(p)
				if len(f) >= 2 {
					sig.params = append(sig.params, param{typ: f[len(f)-2], name: f[len(f)-1]})
				}
			}
			return sig, true
		}
	}
	return signature{}, false
}

type callSite struct {
	at   *tracer
	args []string
}

// callsOf are the calls of method name in the file (its declaration left out), each with its arguments.
func (t *tracer) callsOf(name string) []callSite {
	re := regexp.MustCompile(`(^|[^A-Za-z0-9_.])` + regexp.QuoteMeta(name) + `[[:space:]]*\(`)
	var out []callSite
	for i, ln := range t.text {
		for _, loc := range re.FindAllStringIndex(ln, -1) {
			open := strings.IndexByte(ln[loc[0]:], '(') + loc[0]
			args, ok := balanced(ln, open)
			u, in := at(t.f, i)
			if !ok || !in || u.scope.Start == t.scope.Start {
				continue // the declaration (or a recursive call), or a call not on one line
			}
			out = append(out, callSite{at: u, args: splitArgs(args)})
		}
	}
	return out
}

// balanced is the text inside the parentheses opening at s[open], literal-aware, ok false when they do not
// close on the line.
func balanced(s string, open int) (string, bool) {
	b := javasrc.MaskLine(s)
	depth := 0
	for i := open; i < len(b); i++ {
		switch b[i] {
		case '(':
			depth++
		case ')':
			if depth--; depth == 0 {
				return s[open+1 : i], true
			}
		}
	}
	return "", false
}

// splitArgs splits an argument list at its top-level commas.
func splitArgs(s string) []string {
	b := javasrc.MaskLine(s)
	var out []string
	depth, from := 0, 0
	for i := 0; i < len(b); i++ {
		switch b[i] {
		case '(', '<', '[', '{':
			depth++
		case ')', '>', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(s[from:i]))
				from = i + 1
			}
		}
	}
	if strings.TrimSpace(s) != "" {
		out = append(out, strings.TrimSpace(s[from:]))
	}
	return out
}
