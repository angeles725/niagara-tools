// Package nosystemout is the lint-no-system-out rule: a System.out / System.err print call in module Java
// (it bypasses java.util.logging and the station's Logging service). Port of the awk in
// toolbelt/lint-no-system-out.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins).
package nosystemout

import (
	"regexp"
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

// print is a System.out / System.err print call as a statement: whitespace (newlines included) may sit
// around each dot, and System is a whole word (never MySystem).
var print = regexp.MustCompile(`\bSystem\s*\.\s*(out|err)\s*\.\s*print`)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Check reports every line where a System.out / System.err print call starts, matched over the code
// (linescan.Code: no comment, Javadoc or literal text) joined across lines, once per line.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := strings.Join(linescan.Code(f), "\n")
	var out []lint.Finding
	line, from, last := 1, 0, 0
	for _, m := range print.FindAllStringIndex(code, -1) {
		line += strings.Count(code[from:m[0]], "\n")
		from = m[0]
		if line != last {
			out = append(out, lint.Finding{Line: line, Msg: linescan.Source(f.Raw[line-1])})
			last = line
		}
	}
	return out, nil
}
