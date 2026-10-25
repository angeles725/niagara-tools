// Package privateapiimport is lint-private-api-import (G65): a com.tridium type used by module code. The devguide
// says "com.tridium APIs should never be used by developers, and no compatibility is guaranteed" (apiInfo.txt:54),
// so such a use can break on any Niagara update. The contract is the deferred G65 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-u5-c59-c62.md); toolbelt/lint-private-api-import.sh is the wrapper
// and its header is the user-facing contract.
//
// It is a lint.TreeRule over one module root: every *.java under a -rt, -ux, -wb or -se part's src/, srcTest/,
// test/ and build/ not entered. Java is read as javasrc.Tokens (comments stripped, a literal one token), so an
// import in a comment or a string is not one. A row is each `import com.tridium.…;`, `import static com.tridium.…;`
// and fully qualified `com.tridium.` name in code; com.tridiumx. is another prefix. The default allow-list is the
// NDriver framework the driver wizard generates against (com.tridium.ndriver.** and com.tridium.driver.util.DrUtil,
// retro n4-course-handson-ndriver Δ8); --no-default-allow drops it, --allow <glob> adds to it.
package privateapiimport

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name>
// convention; the deferred spec's row wrote "private-api-import").
const ID = "lint-private-api-import"

// defaultAllow is the NDriver framework: every NDriver module imports it (the course driver: 60 imports).
var defaultAllow = []string{"com.tridium.ndriver.**", "com.tridium.driver.util.DrUtil"}

// Spec registers the rule: [--strict] [--allow <glob>]... [--no-default-allow] <module-root> in any order
// (Mode Valued), WARN rows (exit 1 under --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{
		Tree: &rule{}, Mode: lint.Valued, Options: []string{"--allow"}, Flags: []string{"--no-default-allow"}, ValuedStrict: true, ModuleRoot: true,
		Usage:      "usage: lint-private-api-import.sh [--strict] [--allow <glob>]... [--no-default-allow] <module-root>",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "test", "build"}},
		EmptyError: lint.EmptyRow{Msg: emptyMsg},
	}
}

const emptyMsg = "no *.java under a -rt, -ux, -wb or -se part's src/ (wrong path: pass the module root)"

// rule holds the allow-list Configure built (set); the registered rule, run in a rule list, uses the default one.
type rule struct {
	allow []matcher
	set   bool
}

func (*rule) ID() string              { return ID }
func (*rule) Severity() lint.Severity { return lint.Warn }

// Configure reads --no-default-allow and every --allow glob.
func (*rule) Configure(a lint.Args) (lint.Configured, error) {
	var pats []string
	if !a.Flags["--no-default-allow"] {
		pats = append(pats, defaultAllow...)
	}
	r := &rule{set: true}
	for _, p := range append(pats, a.Values["--allow"]...) {
		r.allow = append(r.allow, compile(p))
	}
	return r, nil
}

var part = regexp.MustCompile(`(^|/)[^/]+-(rt|ux|wb|se)/src/`)

// CheckTree reads every part source's tokens.
func (r *rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	allow := r.allow
	if !r.set {
		for _, p := range defaultAllow {
			allow = append(allow, compile(p))
		}
	}
	var out []lint.Finding
	read := false
	for _, f := range files {
		if !part.MatchString(f.Path) {
			continue
		}
		read = true
		for _, u := range uses(javasrc.Tokens(f.Stripped())) {
			if !allowed(allow, u.name) {
				out = append(out, lint.Finding{Path: f.Path, Line: u.line + 1,
					Msg: u.name + " is private API (devguide apiInfo: no compatibility guaranteed)"})
			}
		}
	}
	if !read {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

type use struct {
	name string
	line int // 0-based
}

// uses lists each com.tridium name of an import statement and each fully qualified com.tridium name in code; a
// package statement is skipped.
func uses(toks []javasrc.Token) []use {
	var out []use
	for k := 0; k < len(toks); k++ {
		t := toks[k]
		switch {
		case t.Kind == javasrc.TokIdent && (t.Text == "import" || t.Text == "package") && (k == 0 || toks[k-1].Text == ";" || toks[k-1].Text == "}"):
			j := k + 1
			if j < len(toks) && toks[j].Text == "static" {
				j++
			}
			var b strings.Builder
			for ; j < len(toks) && toks[j].Text != ";"; j++ {
				b.WriteString(toks[j].Text)
			}
			if name := b.String(); t.Text == "import" && strings.HasPrefix(name, "com.tridium.") {
				out = append(out, use{name, t.Line})
			}
			k = j
		case t.Text == "com" && (k == 0 || toks[k-1].Text != ".") && k+2 < len(toks) && toks[k+1].Text == "." && toks[k+2].Text == "tridium":
			name, j := "com", k+1
			for ; j+1 < len(toks) && toks[j].Text == "." && toks[j+1].Kind == javasrc.TokIdent; j += 2 {
				name += "." + toks[j+1].Text
			}
			if name != "com.tridium" {
				out = append(out, use{strings.TrimSuffix(name, ".class"), t.Line})
			}
			k = j - 1
		}
	}
	return out
}

// matcher is one allow-list entry: a glob ("*" one name segment, "**" any) or an exact name, which also
// allows its members (a static import or a call through the type).
type matcher struct {
	exact string
	glob  *regexp.Regexp
}

func compile(p string) matcher {
	if !strings.Contains(p, "*") {
		return matcher{exact: p}
	}
	var b strings.Builder
	for i := 0; i < len(p); i++ {
		switch {
		case strings.HasPrefix(p[i:], "**"):
			b.WriteString(".*")
			i++
		case p[i] == '*':
			b.WriteString(`[^.]*`)
		default:
			b.WriteString(regexp.QuoteMeta(p[i : i+1]))
		}
	}
	return matcher{glob: regexp.MustCompile("^" + b.String() + "$")}
}

func (m matcher) allows(name string) bool {
	if m.glob != nil {
		return m.glob.MatchString(name)
	}
	return name == m.exact || strings.HasPrefix(name, m.exact+".")
}

func allowed(ms []matcher, name string) bool {
	for _, m := range ms {
		if m.allows(name) {
			return true
		}
	}
	return false
}
