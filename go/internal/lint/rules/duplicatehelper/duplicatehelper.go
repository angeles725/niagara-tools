// Package duplicatehelper is lint-duplicate-helper (G41): logic pasted as a second copy instead of being
// called or extracted into one shared helper. The contract is the deferred G41 spec
// (build-n4-module-kit/ledger.d/2026-10-22-modular-line-efficient-quality.md); the doctrine is METHODOLOGY.md
// § rt "One shared helper per concern, before a second copy lands"; toolbelt/lint-duplicate-helper.sh is the
// wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule: every *.java under a -rt, -ux or -wb part's src/ of one module root is compared
// with every other. The part is matched on the listed path, so a part's own src/ given as the root is
// compared alone, and a GROUP directory given as the root compares its sibling modules too. Methods come from javasrc.Decls on javasrc.Tokens (comments stripped, a literal or a
// text block one token), so a copy in a comment or a text block is never code; a method whose name line
// lies in a slotomatic BEGIN/END BAJA AUTO GENERATED CODE region is skipped. A later definition (walk
// order, then source order) is a row when
//   - its body has the same javasrc.Shape (whitespace and comments dropped, identifiers and literals
//     kept) as an earlier body of at least minStatements javasrc.Statements, or else
//   - it is a helper (private or static: neither can override) sharing its name and parameter types with
//     an earlier helper of another class, neither being an @Override, an anonymous-class method or a
//     constructor. Left out (2026-10-24 real-tree tunings, each with a test): a public or package
//     instance method (it may implement a supertype's method without @Override: TableModel getters, a
//     command's doInvoke), a static factory returning its own class, and a one-statement delegate that
//     calls the same-named method on another type (the shared helper's caller).
//
// The row names the first definition only, so N copies give N-1 rows.
package duplicatehelper

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name>
// convention; the deferred spec's row wrote "duplicate-helper").
const ID = "lint-duplicate-helper"

// minStatements is the spec's floor for a body-shape match.
const minStatements = 8

// Spec registers the rule: [--strict] <module-root>... (WARN rows: exit 0, 1 under --strict; each root is
// compared on its own), *.java outside srcTest/ and build/, and a root with no part src/ file is the
// ERROR row (exit 3). Its roots are module roots, so n4lint --all leaves it out (Spec.ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-duplicate-helper.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "build"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt, -ux or -wb part's src/ (wrong path: pass the module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// part is a file under <part>-rt|-ux|-wb/src/.
var part = regexp.MustCompile(`(^|/)[^/]+-(rt|ux|wb)/src/`)

const (
	regionBegin = "BEGIN BAJA AUTO GENERATED CODE"
	regionEnd   = "END BAJA AUTO GENERATED CODE"
)

// site is where a definition is: its file and its 1-based name line.
type site struct {
	path string
	line int
}

func (s site) String() string { return s.path + ":" + strconv.Itoa(s.line) }

// CheckTree compares every part file's method bodies and signatures in walk order.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	shapes := map[string]site{}
	sigs := map[string]site{}
	var out []lint.Finding
	read := false
	for _, f := range files {
		if !part.MatchString(f.Path) {
			continue
		}
		read = true
		gen := generated(f)
		toks := javasrc.Tokens(f.Stripped())
		for _, d := range javasrc.Decls(toks) {
			if gen[d.Line] {
				continue
			}
			at := site{f.Path, d.Line + 1}
			row := func(first site, detail string) {
				out = append(out, lint.Finding{Path: at.path, Line: at.line, Msg: "same shape as " + first.String() + " (" + detail + ")"})
			}
			body := toks[d.Body[0]:d.Body[1]]
			matched := false
			if n := javasrc.Statements(body); n >= minStatements {
				shape := javasrc.Shape(body)
				if first, ok := shapes[shape]; ok {
					row(first, d.Name+": same body, "+strconv.Itoa(n)+" statements")
					matched = true
				} else {
					shapes[shape] = at
				}
			}
			if !d.Private && !d.Static || d.Override || d.Anonymous || d.Name == d.Class || d.Result == d.Class ||
				delegates(body, d.Name) {
				continue // not a helper copy: it may implement a supertype's method, construct its own type or call the helper
			}
			sig := d.Name + "(" + strings.Join(d.Params, ",") + ")"
			first, ok := sigs[sig]
			switch {
			case !ok:
				sigs[sig] = at
			case !matched: // one class cannot declare a signature twice, so the first is another class's
				row(first, sig+": same name and parameter types")
			}
		}
	}
	if !read {
		return nil, lint.ErrEmptyRoot
	}
	return out, nil
}

// delegates reports a one-statement body that calls name on another type (Other.name(...)): the caller of a
// shared helper, which is the fix the rule asks for, not a copy.
func delegates(body []javasrc.Token, name string) bool {
	if javasrc.Statements(body) != 1 {
		return false
	}
	for j := 1; j+1 < len(body); j++ {
		if body[j].Text == name && body[j-1].Text == "." && body[j+1].Text == "(" {
			return true
		}
	}
	return false
}

// generated marks the 0-based lines inside slotomatic regions: from a line whose comment holds the BEGIN
// marker to the line whose comment holds the END marker (a marker inside a string literal is code).
func generated(f *javasrc.File) map[int]bool {
	gen := map[int]bool{}
	in := false
	for i, raw := range f.Raw {
		inComment := func(m string) bool { return strings.Contains(raw, m) && !strings.Contains(f.Stripped()[i], m) }
		switch {
		case inComment(regionBegin):
			in = true
		case inComment(regionEnd):
			gen[i], in = true, false
		}
		if in {
			gen[i] = true
		}
	}
	return gen
}
