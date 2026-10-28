// Package loggername is lint-logger-name (G40): a java.util.logging logger whose name is not the module's moduleName.
// types/observability.md § 1 makes the logger name the exact moduleName (or a <moduleName>.<child> logger), the name
// the DebugService and Logger Configuration list. The contract is the deferred G40 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-process.md, C52); toolbelt/lint-logger-name.sh is the wrapper and its
// header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a -rt, -ux, -wb or -se part's src/ (srcTest/ and a
// part's own test/ and build/ not entered; a package named test or build under src/ is read) and every *.gradle /
// *.gradle.kts. A part's module name is the moduleName.set("X") or moduleName = "X" of a gradle file in the part
// directory, else of one in the directory above it (the group), else the part directory name without its
// -rt/-ux/-wb/-se suffix. A row is each Logger.getLogger("<literal>") whose literal is neither the module name nor
// <module name>.<child>, and each Logger.getLogger(X.class.getName()). Java is read as javasrc tokens, so a comment or
// a string is never a call; a non-literal argument is not judged.
package loggername

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column.
const ID = "lint-logger-name"

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), WARN rows (exit 1 under
// --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-logger-name.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java", "*.gradle", "*.gradle.kts"}, Prune: []string{"srcTest"}, PrunePaths: partOutputs},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt, -ux, -wb or -se part's src/ (wrong path: pass the module root)"}}
}

// partOutputs are a part's own test/ and build/ directories, never entered. They are pruned by path, not by name: a
// Java package named test or build under src/ (com/acme/test/) is source and is read (run-3 review R3).
var partOutputs = []string{"*-rt/test", "*-rt/build", "*-ux/test", "*-ux/build", "*-wb/test", "*-wb/build", "*-se/test", "*-se/build"}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	part       = regexp.MustCompile(`^(.*/)?([^/]+)-(rt|ux|wb|se)/src/`)
	moduleName = regexp.MustCompile(`\bmoduleName\s*(?:\.set\(\s*|=\s*)"([^"]+)"`)
)

// CheckTree reads the module names the gradle files declare, then every part source's getLogger calls.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	declared := map[string]string{} // directory -> moduleName of a gradle file in it
	for _, f := range files {
		if strings.HasSuffix(f.Path, ".gradle") || strings.HasSuffix(f.Path, ".gradle.kts") {
			if m := moduleName.FindStringSubmatch(strings.Join(f.Raw, "\n")); m != nil {
				declared[dir(f.Path)] = m[1]
			}
		}
	}
	var out []lint.Finding
	read := false
	for _, f := range files {
		m := part.FindStringSubmatch(f.Path)
		if m == nil || !strings.HasSuffix(f.Path, ".java") {
			continue
		}
		read = true
		partDir := m[1] + m[2] + "-" + m[3]
		name, ok := declared[partDir]
		if !ok {
			if name, ok = declared[strings.TrimSuffix(m[1], "/")]; !ok {
				name = m[2]
			}
		}
		for _, c := range calls(javasrc.Tokens(f.Stripped())) {
			if msg := judge(c, name); msg != "" {
				out = append(out, lint.Finding{Path: f.Path, Line: c.line + 1, Msg: msg})
			}
		}
	}
	if !read {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

func dir(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return "."
}

// call is one Logger.getLogger(...) call: its argument as written and whether it is one string literal.
type call struct {
	arg     string
	literal string // the literal's value when the argument is one string literal
	class   bool   // the argument is X.class.getName()
	line    int
}

func calls(toks []javasrc.Token) []call {
	var out []call
	for k := 2; k+1 < len(toks); k++ {
		if toks[k].Text != "getLogger" || toks[k-1].Text != "." || toks[k-2].Text != "Logger" || toks[k+1].Text != "(" {
			continue
		}
		depth, end := 0, len(toks)
		for j := k + 1; j < len(toks); j++ {
			if toks[j].Text == "(" {
				depth++
			} else if toks[j].Text == ")" {
				if depth--; depth == 0 {
					end = j
					break
				}
			}
		}
		arg := toks[k+2 : end]
		var b strings.Builder
		for _, t := range arg {
			b.WriteString(t.Text)
		}
		c := call{arg: b.String(), line: toks[k].Line}
		switch {
		case len(arg) == 1 && arg[0].Kind == javasrc.TokLiteral && strings.HasPrefix(arg[0].Text, `"`):
			c.literal = strings.Trim(arg[0].Text, `"`)
		case strings.HasSuffix(c.arg, ".class.getName()"):
			c.class = true
		default:
			continue
		}
		out = append(out, c)
	}
	return out
}

func judge(c call, module string) string {
	switch {
	case c.class:
		return "Logger.getLogger(" + c.arg + ") names a class, not the module " + module + " (types/observability.md § 1)"
	case c.literal == module || strings.HasPrefix(c.literal, module+"."):
		return ""
	}
	return "Logger.getLogger(\"" + c.literal + "\") is not the module name " + module + " or a " + module + ".<child> logger (types/observability.md § 1)"
}
