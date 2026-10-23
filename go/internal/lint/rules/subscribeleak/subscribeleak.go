// Package subscribeleak is the lint-subscribe-without-unsubscribe rule: a file that calls `.subscribe(`
// and never unsubscribes (a Subscriber leak). Port of the check in
// toolbelt/lint-subscribe-without-unsubscribe.sh (the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins).
package subscribeleak

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-subscribe-without-unsubscribe"

// Spec registers the rule with the argument contract of toolbelt/lint-subscribe-without-unsubscribe.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-subscribe-without-unsubscribe.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_strip_run (awk_lint_list), which scans the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports the first `.subscribe(` line when no line holds `unsubscribe`, both in the code
// before each line's first "//".
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	at := 0
	for i, ln := range f.Raw {
		code := linescan.Cut(ln)
		if strings.Contains(code, "unsubscribe") {
			return nil, nil
		}
		if at == 0 && strings.Contains(code, ".subscribe(") {
			at = i + 1
		}
	}
	if at == 0 {
		return nil, nil
	}
	return []lint.Finding{{Line: at, Msg: linescan.Trimmed(f.Raw[at-1])}}, nil
}
