// Package bformatcall is lint-bformat-call: BFormat special calls written in a form BFormat never parses.
// BFormat matches time() and user() only as the exact calls "time()" and "user()" (S/util/BFormat.java:443-444), so
// %time(<arg>)% and %user% fall through to reflection and print %err:...%. Port of toolbelt/lint-bformat-call.sh (the
// wrapper keeps the user-facing contract and its header documents the rule and its mutation pins).
//
// It is a per-file lint.Rule over *.java (javasrc.Stripped: comments blanked, string literals kept, since BFormat
// patterns live in strings), *.lexicon, *.bog and *.palette (read as text).
package bformatcall

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-bformat-call"

// Spec registers the rule with the argument contract of toolbelt/lint-bformat-call.sh: [--strict] [--] <src-root>,
// the awk_lint_run walk (a listing error still scans the files listed, then exit 3) and the K20 row.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-bformat-call.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java", "*.lexicon", "*.bog", "*.palette"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	timeArg  = regexp.MustCompile(`%time\([^)]`)
	bareUser = regexp.MustCompile(`%user%`)
)

const (
	bfc1 = "BFC1 time() takes no argument -- %time(<fmt>)% goes to reflection and prints %err:...%; chain instead: %time().toDateString% (types/logic.md BFormat)"
	bfc2 = "BFC2 %user% is a reflective call that fails -- write %user()% (types/logic.md BFormat)"
)

// Check is the awk END block: per line, a BFC1 row, then a BFC2 row.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := f.Raw
	if strings.HasSuffix(f.Path, ".java") {
		lines = f.Stripped()
	}
	return scan(lines, func(i int) int { return i + 1 }), nil
}

// scan reports each text's BFC1 and BFC2 rows at line(i).
func scan(texts []string, line func(i int) int) []lint.Finding {
	var out []lint.Finding
	for i, t := range texts {
		if timeArg.MatchString(t) {
			out = append(out, lint.Finding{Line: line(i), Msg: bfc1})
		}
		if bareUser.MatchString(t) {
			out = append(out, lint.Finding{Line: line(i), Msg: bfc2})
		}
	}
	return out
}
