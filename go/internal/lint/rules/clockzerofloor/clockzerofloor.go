// Package clockzerofloor is the lint-clock-zero-floor rule: a Math.max(0, delay) floor in a file that
// calls Clock.schedule (a schedule time <= 0 throws, so the floor must be 1 ms). Port of the awk in
// toolbelt/lint-clock-zero-floor.sh (the wrapper keeps the user-facing contract and its header documents
// the rule and its mutation pins).
package clockzerofloor

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-clock-zero-floor"

// Spec registers the rule with the argument contract of toolbelt/lint-clock-zero-floor.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-clock-zero-floor.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

var zeroFloor = regexp.MustCompile(`Math\.max\([[:space:]]*0[Ll]?[[:space:]]*,`)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports every zero floor in the code (linescan.Code: no comment, Javadoc or literal text), only
// when the file calls Clock.schedule in its code.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	sched := false
	for i, code := range linescan.Code(f) {
		sched = sched || strings.Contains(code, "Clock.schedule")
		if zeroFloor.MatchString(code) {
			out = append(out, lint.Finding{Line: i + 1, Msg: "zero-floor: " + linescan.Source(f.Raw[i])})
		}
	}
	if !sched {
		return nil, nil
	}
	return out, nil
}
