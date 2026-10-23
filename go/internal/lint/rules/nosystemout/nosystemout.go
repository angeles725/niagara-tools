// Package nosystemout is the lint-no-system-out rule: a System.out / System.err print call in module Java
// (it bypasses java.util.logging and the station's Logging service). Port of the awk in
// toolbelt/lint-no-system-out.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins).
package nosystemout

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-no-system-out"

// Spec registers the rule with the argument contract of toolbelt/lint-no-system-out.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-no-system-out.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Check reports every line whose code before its first "//" holds System.out.print or System.err.print.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i, ln := range f.Raw {
		if code := linescan.Cut(ln); strings.Contains(code, "System.out.print") || strings.Contains(code, "System.err.print") {
			out = append(out, lint.Finding{Line: i + 1, Msg: linescan.Source(ln)})
		}
	}
	return out, nil
}
