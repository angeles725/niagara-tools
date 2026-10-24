// Package persisthotwrite is the lint-persist-hot-write rule: a non-transient slot-o-matic property's
// setter called from changed(Property ...) or from any zero-arg method changed() calls (one hop), with no
// cadence guard (a Clock.millis( or a `% n ==/!=` test) on that line or the 5 above it in the same body.
// Port of the awk in toolbelt/lint-persist-hot-write.sh (the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins); it shares changed(), bodies, callees and properties
// with lint-changed-hot-write through rules/hotwrite.
package persisthotwrite

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/hotwrite"
)

// ID is the lint name printed in every row.
const ID = "lint-persist-hot-write"

// Spec registers the rule with the argument contract of toolbelt/lint-persist-hot-write.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-persist-hot-write.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	setterDecl = regexp.MustCompile(`(void|public|private|protected|static)[[:space:]]+[A-Za-z_<>\[\],. ]*set[A-Z]`)
	modulo     = regexp.MustCompile(`%[[:space:]]*[A-Za-z0-9_]+[[:space:]]*(==|!=)`)
)

type body struct{ start, end int } // 0-based lines

// Check WARNs every unguarded setter call of a non-transient property in changed() or a one-hop callee,
// property by property, body by body (changed() first), line by line.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := hotwrite.View(f)
	start, _, ok := hotwrite.Changed(lines)
	if !ok {
		return nil, nil
	}
	end, ok := hotwrite.BodyEnd(lines, start)
	if !ok {
		return nil, nil
	}
	bodies := []body{{start, end}}
	for _, name := range hotwrite.Callees(lines, start, end) {
		if d, ok := hotwrite.Decl(lines, name); ok {
			if e, ok := hotwrite.BodyEnd(lines, d); ok {
				bodies = append(bodies, body{d, e})
			}
		}
	}
	var out []lint.Finding
	for _, prop := range hotwrite.NonTransient(lines) {
		setter := hotwrite.Setter(prop) + "("
		for _, b := range bodies {
			for i := b.start; i <= b.end; i++ {
				if !strings.Contains(lines[i], setter) || setterDecl.MatchString(lines[i]) || guarded(lines, b.start, i) {
					continue
				}
				out = append(out, lint.Finding{Line: i + 1, Msg: setter + "...) writes non-transient slot '" + prop +
					"' reached from changed(), no cadence guard (Clock.millis()/counter) within 5 lines --" +
					" integrate in memory, checkpoint on a periodic tick + in stopped(), seed once on start"})
			}
		}
	}
	return out, nil
}

// guarded reports a cadence guard on line i or the 5 above it, not above the body's first line.
func guarded(lines []string, start, i int) bool {
	for j := i; j >= start && j >= i-5; j-- {
		if strings.Contains(lines[j], "Clock.millis(") || modulo.MatchString(lines[j]) {
			return true
		}
	}
	return false
}
