// Package bqlconcat is the lint-bql-string-concat rule: a line that builds a bql: ORD or a
// BqlQuery.make(...) by string concatenation (an injection risk; BQL has no parameterized query).
// Port of the awk in toolbelt/lint-bql-string-concat.sh (the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins).
package bqlconcat

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-bql-string-concat"

// Spec registers the rule with the argument contract of toolbelt/lint-bql-string-concat.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-bql-string-concat.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

var (
	bql    = regexp.MustCompile(`bql:|BqlQuery\.make\(`)
	concat = regexp.MustCompile(`"[[:space:]]*\+|\+[[:space:]]*"`)
)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports every line that names a bql: ORD or BqlQuery.make( and has a concatenation operator
// next to a quote, both in the code before its first "//".
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i, ln := range f.Raw {
		if code := linescan.Cut(ln); bql.MatchString(code) && concat.MatchString(code) {
			out = append(out, lint.Finding{Line: i + 1, Msg: "bql concat: " + linescan.Source(ln)})
		}
	}
	return out, nil
}
