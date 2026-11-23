package bql

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Retro mcpn4-status-isok Δ2: a BQL where-clause that a station resolves to nothing is an empty
// answer, never an error, and the fake station of the tests accepts any string, so no offline test
// can tell a predicate the station understands from one it silently drops (`status.ok = false`
// matched nothing on a live 4.14 while every test was green). The rule that replaces the missing
// check: a source file that ships a BQL predicate says, in a comment, the station version it was
// live-checked on. This test holds that rule for the package.

// predicate matches a literal that is a BQL where-clause or a bare field comparison.
var predicate = regexp.MustCompile(`(?i)\bwhere\s+[\w.]+\s*(=|!=|<|>|like\b)|^[\w.]+\s*(=|!=)\s*(true|false)$`)

// liveNote matches the note: "live-checked on 4.14.0.162".
var liveNote = regexp.MustCompile(`(?i)live-checked on \d+\.\d+\.\d+`)

// unnoted lists, per Go source file (tests excluded) under dir, the predicate literals of a file
// that has no live-check note in any comment.
func unnoted(t *testing.T, dir string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		noted := false
		for _, g := range file.Comments {
			noted = noted || liveNote.MatchString(g.Text())
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING || noted {
				return true
			}
			if s, err := strconv.Unquote(lit.Value); err == nil && predicate.MatchString(s) {
				out = append(out, filepath.Base(f)+": "+s)
			}
			return true
		})
	}
	return out
}

func TestEveryBqlPredicateInThePackageCarriesALiveCheckNote(t *testing.T) {
	if got := unnoted(t, "."); len(got) > 0 {
		t.Errorf("BQL predicates in a file with no 'live-checked on <station version>' comment "+
			"(retro mcpn4-status-isok Δ2): %q", got)
	}
}

// The check itself: a predicate without the note is found, with it is not.
func TestTheLiveCheckRuleSeesAPredicateWithoutANote(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("bare.go", "package bql\nconst q = \"select a where status.ok = false\"\nconst w = \"status.fault = true\"\n")
	write("noted.go", "package bql\n// Live-checked on 4.14.0.162 (2026-11-19).\nconst q = \"select a where status.isOk = false\"\n")
	write("plain.go", "package bql\nconst q = \"select a from control:ControlPoint\"\n")
	write("skip_test.go", "package bql\nconst q = \"select a where status.ok = false\"\n")
	got := unnoted(t, dir)
	if len(got) != 2 || !strings.HasPrefix(got[0], "bare.go: ") || !strings.HasPrefix(got[1], "bare.go: ") {
		t.Errorf("unnoted = %q, want the two predicates of bare.go only", got)
	}
}
