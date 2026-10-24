// Package machinedefaultpath is lint-machine-default-path (G43): a default value in Java source that is an
// absolute machine path, so the module only works on the machine (or in the session) it was written on.
// The doctrine is METHODOLOGY.md § rt "No machine- or session-specific default paths in source" and
// types/issues-and-gotchas.md I3; toolbelt/lint-machine-default-path.sh is the wrapper and its header is the
// user-facing contract.
//
// A default value is a "..." literal in one of three places, read on the shared scanner's blanked code
// (comments out, literal contents removed), so comment and Javadoc text never counts:
//   - an @NiagaraProperty defaultValue: the inner literals of its expression (BString.make("...")), or the
//     whole text when it calls nothing;
//   - a field initializer: the literal right after "<type> <name> =" on a line outside every method body;
//   - a fallback argument: the literal right after the "," of getProperty(k, / getOrDefault(k, /
//     requireNonNullElse(x, (one level of nested calls in the first argument), or right after orElse(.
//
// The text before the literal may start on up to three earlier lines ("String D =\n \"...\"").
// Only the first literal of a concatenation is read: "/home/" + user is a row, "/ho" + "me" is not.
package machinedefaultpath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints.
const ID = "lint-machine-default-path"

// Spec registers the rule: [--strict] <src-root>... (--strict changes nothing: every row is ADVISORY), the
// *.java sources outside srcTest/ (test fixtures may name a scratch path), and the engine's empty-root ERROR.
func Spec() lint.Spec {
	return lint.Spec{Rule: rule{}, Mode: lint.Each, Usage: "usage: lint-machine-default-path.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest"}}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Advisory }

// The default contexts, matched on the blanked code before a literal (trailing space trimmed).
var (
	slotDefault = regexp.MustCompile(`\bdefaultValue\s*=$`)
	fieldInit   = regexp.MustCompile(`[\w$>\]]\s+[A-Za-z_$][\w$]*\s*=$`)
	fallback    = regexp.MustCompile(`(\b(getProperty|getOrDefault|requireNonNullElse)\s*\(([^()]|\([^()]*\))*,|\borElse\s*\()$`)
	// machine is an absolute machine path: a home, a temporary directory, a WSL or Windows user directory.
	// /home/niagara/ is left out: the JACE platform's own account home, the same on every unit.
	machine = regexp.MustCompile(`^(/home/|/Users/|/(var/)?tmp(/|$)|/mnt/[A-Za-z]/Users/|/?[A-Za-z]:[\\/]+(?i:users)[\\/])`)
)

const platformHome = "/home/niagara/"

const lookBack = 3 // earlier lines the text before a literal may start on

// Check reports the first machine path per line among the file's default literals.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var lits []javasrc.Literal
	// A '...' char literal is never a path, so every literal is classified.
	code, _ := javasrc.BlankLiterals(f.Stripped(), func(l javasrc.Literal) { lits = append(lits, l) })
	inBody := make([]bool, len(code))
	for _, m := range f.Methods() {
		for i := m.Start - 1; i < m.End && i < len(code); i++ {
			inBody[i] = true
		}
	}
	var out []lint.Finding
	last := -1
	for _, l := range lits {
		if l.Line == last {
			continue
		}
		where, p := "", ""
		switch pre := before(code, l); {
		case slotDefault.MatchString(pre):
			where, p = "an @NiagaraProperty defaultValue", exprPath(unescape(l.Text))
		case fallback.MatchString(pre):
			where, p = "a fallback argument", path(unescape(l.Text))
		case !inBody[l.Line] && fieldInit.MatchString(pre):
			where, p = "a field initializer", path(unescape(l.Text))
		}
		if p == "" {
			continue
		}
		last = l.Line
		out = append(out, lint.Finding{Line: l.Line + 1, Msg: `machine-specific default path "` + p + `" in ` + where +
			" -- take it from a slot, a station-relative ORD, an environment variable or the site table (METHODOLOGY.md § rt)"})
	}
	return out, nil
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
	if (machine.MatchString(p) && !strings.HasPrefix(p, platformHome)) || strings.Contains(strings.ToLower(s), "scratchpad") {
		return s
	}
	return ""
}

// unescape reads the Java escapes a path literal uses (\\ and \"); any other escape is kept as written.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '\\' || s[i+1] == '"') {
			i++
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
