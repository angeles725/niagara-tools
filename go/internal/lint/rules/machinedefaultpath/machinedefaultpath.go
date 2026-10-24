// Package machinedefaultpath is lint-machine-default-path (G43): a default value that is an absolute machine
// path, so the module only works on the machine (or in the session) it was written on. The contract is the
// deferred G43 spec (build-n4-module-kit/ledger.d/2026-10-22-modular-line-efficient-quality.md); the doctrine
// is METHODOLOGY.md § rt "No machine- or session-specific default paths in source" and
// types/issues-and-gotchas.md I3; toolbelt/lint-machine-default-path.sh is the wrapper and its header is the
// user-facing contract.
//
// A default value is, in a *.java file, a "..." literal read on the shared scanner's blanked code (comments
// out, literal contents removed), so comment and Javadoc text never counts:
//   - an @NiagaraProperty defaultValue: the inner literals of its expression (BString.make("...")), or the
//     whole text when it calls nothing;
//   - a field initializer: the literal right after "<type> <name> =" on a line outside every method body;
//   - a fallback argument: the literal right after the "," of getProperty(k, / getOrDefault(k, /
//     requireNonNullElse(x, (one level of nested calls in the first argument), or right after orElse(.
//
// The text before the literal may start on up to three earlier lines ("String D =\n \"...\"").
// Only the first literal of a concatenation is read: "/home/" + user is a row, "/ho" + "me" is not.
//
// In a *.js file under an rc/ directory (jssrc tokens, comments dropped), a string or a template literal with
// no ${...} that is a const/let/var initializer (`const X = "..."`, at any depth: rc code lives in IIFE and
// AMD wrappers, so a top-level-only rule would read nothing) or an object-literal property default
// (`{ key: "..." }`). In a *.properties file (props.Read), every value, except that a gradle.properties
// key in props.BuildLocationKeys gets no row: lint-structure L10 owns those (R10-D8), in kit fixtures too.
package machinedefaultpath

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/props"
)

// ID is the lint name n4lint accepts and every row prints.
const ID = "lint-machine-default-path"

// Spec registers the rule: [--strict] <module-root>... (WARN rows: exit 0, 1 under --strict), the *.java,
// *.js and *.properties files outside srcTest/ (test sources), build/ (Gradle output, unpacked jars) and
// node_modules/ (third-party), and the engine's empty-root ERROR.
func Spec() lint.Spec {
	return lint.Spec{Rule: rule{}, Mode: lint.Each, Usage: "usage: lint-machine-default-path.sh [--strict] <module-root>...",
		Select: lint.Selection{Names: []string{"*.java", "*.js", "*.properties"}, Prune: []string{"srcTest", "build", "node_modules"}},
		// The engine's default text for these Names, stated so a reader (and rules_test, which assumes the
		// *.java default for a rule without EmptyError) sees the three file kinds.
		EmptyError: lint.EmptyRow{Msg: "no *.java or *.js or *.properties sources found (wrong path or empty source tree)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// The default contexts, matched on the blanked code before a literal (trailing space trimmed).
var (
	slotDefault = regexp.MustCompile(`\bdefaultValue\s*=$`)
	fieldInit   = regexp.MustCompile(`[\w$>\]]\s+[A-Za-z_$][\w$]*\s*=$`)
	fallback    = regexp.MustCompile(`(\b(getProperty|getOrDefault|requireNonNullElse)\s*\(([^()]|\([^()]*\))*,|\borElse\s*\()$`)
	// machine is an absolute machine path: a home, a temporary directory, a WSL or Windows user directory,
	// case-insensitive as the spec reads it. /home/niagara/ is left out: the JACE platform's own account
	// home, the same on every unit.
	machine = regexp.MustCompile(`(?i)^(/home/|/Users/|/(var/)?tmp(/|$)|/mnt/[A-Za-z]/Users/|/?[A-Za-z]:[\\/]+users[\\/])`)
)

const platformHome = "/home/niagara/"

const lookBack = 3 // earlier lines the text before a literal may start on

// Check reports the first machine path per line among the file's default values.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	switch {
	case strings.HasSuffix(f.Path, ".properties"):
		return properties(f), nil
	case strings.HasSuffix(f.Path, ".js"):
		if strings.Contains(f.Path, "/rc/") {
			return script(f), nil
		}
		return nil, nil
	}
	return java(f), nil
}

// finding is the row for path p found in where on 0-based line i.
func finding(i int, p, where string) lint.Finding {
	return lint.Finding{Line: i + 1, Msg: `machine-specific default "` + p + `" in ` + where +
		" -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)"}
}

// perLine keeps the first finding per line.
type perLine struct {
	out  []lint.Finding
	last int
}

func (r *perLine) add(i int, p, where string) {
	if p != "" {
		r.push(i, finding(i, p, where))
	}
}

func (r *perLine) push(i int, fd lint.Finding) {
	if len(r.out) == 0 || i != r.last {
		r.out, r.last = append(r.out, fd), i
	}
}

// properties is every value of a .properties file but a gradle.properties build-location key, which is
// lint-structure L10's (R10-D8: WARN to migrate it to the user-level file; FAIL in a kit fixture, K16).
func properties(f *javasrc.File) []lint.Finding {
	gradle := filepath.Base(f.Path) == "gradle.properties"
	var r perLine
	for _, e := range props.Read(f.Raw) {
		if !(gradle && props.BuildLocationKey(e.Key)) {
			r.add(e.Line, path(e.Value), "a .properties value")
		}
	}
	return r.out
}

// script is the const/let/var initializers and object-literal property defaults of an rc JavaScript file.
func script(f *javasrc.File) []lint.Finding {
	var r perLine
	toks := jssrc.Tokens(f.Raw)
	text := func(k int, s string) bool { return k >= 0 && toks[k].Kind == jssrc.Punct && toks[k].Text == s }
	for k := 3; k < len(toks); k++ {
		t := toks[k]
		if t.Kind != jssrc.String && (t.Kind != jssrc.Template || strings.Contains(t.Text, "${")) {
			continue
		}
		switch name, kw := toks[k-2], toks[k-3]; {
		case text(k-1, "=") && name.Kind == jssrc.Ident && kw.Kind == jssrc.Ident &&
			(kw.Text == "const" || kw.Text == "let" || kw.Text == "var"):
			r.add(t.Line, path(unescape(t.Text)), "a const/let/var initializer")
		case text(k-1, ":") && (name.Kind == jssrc.Ident || name.Kind == jssrc.String) && (text(k-3, "{") || text(k-3, ",")):
			r.add(t.Line, path(unescape(t.Text)), "an object-literal property default")
		}
	}
	return r.out
}

// java is the defaultValue, field-initializer and fallback literals of a Java file.
func java(f *javasrc.File) []lint.Finding {
	var lits []javasrc.Literal
	// A '...' char literal is never a path, so every literal is classified.
	code, _ := javasrc.BlankLiterals(f.Stripped(), func(l javasrc.Literal) { lits = append(lits, l) })
	inBody := make([]bool, len(code))
	for _, m := range f.Methods() {
		for i := m.Start - 1; i < m.End && i < len(code); i++ {
			inBody[i] = true
		}
	}
	var r perLine
	for _, l := range lits {
		switch pre := before(code, l); {
		case slotDefault.MatchString(pre):
			r.add(l.Line, exprPath(unescape(l.Text)), "an @NiagaraProperty defaultValue")
		case fallback.MatchString(pre):
			r.add(l.Line, path(unescape(l.Text)), "a fallback argument")
		case !inBody[l.Line] && fieldInit.MatchString(pre):
			r.add(l.Line, path(unescape(l.Text)), "a field initializer")
		}
	}
	return r.out
}

// before is the blanked code before l's opening quote, reaching back over blank lines, right-trimmed.
func before(code []string, l javasrc.Literal) string {
	pre := code[l.Line][:l.Col-1]
	for i := l.Line - 1; i >= 0 && l.Line-i <= lookBack && strings.TrimSpace(pre) == ""; i-- {
		pre = code[i] + "\n" + pre
	}
	return strings.TrimRight(pre, " \t\r\n")
}

// exprPath is the first machine path among the inner literals of a defaultValue expression, or the whole
// text's when the expression calls nothing ("BString.make(Scratchpad.DIR)" names no path).
func exprPath(expr string) string {
	if !strings.Contains(expr, "(") {
		return path(expr)
	}
	found := ""
	javasrc.BlankLiterals([]string{expr}, func(l javasrc.Literal) {
		if found == "" {
			found = path(unescape(l.Text))
		}
	})
	return found
}

// path returns s when it is a machine path (after an optional file: scheme) or names a scratchpad, else "".
func path(s string) string {
	p := s
	if rest, ok := strings.CutPrefix(p, "file:"); ok && strings.HasPrefix(rest, "/") {
		p = "/" + strings.TrimLeft(rest, "/")
	}
	if (machine.MatchString(p) && !strings.HasPrefix(strings.ToLower(p), platformHome)) || strings.Contains(strings.ToLower(s), "scratchpad") {
		return s
	}
	return ""
}

// unescape reads the escapes a path literal uses (\\, \" and, in JavaScript, \' and \`); any other escape is
// kept as written.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && strings.IndexByte(`\"'`+"`", s[i+1]) >= 0 {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
