// Package arbitraryord is the lint-arbitrary-ord rule: BOrd.make(<lowercase identifier>), an ORD resolved
// from a variable that may hold client input, unless a `// lint-arbitrary-ord: reviewed <reason>` marker
// sits on the line or on the comment line above. Port of the awk in toolbelt/lint-arbitrary-ord.sh (the
// wrapper keeps the user-facing contract; its header documents the rule and its mutation pins).
package arbitraryord

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-arbitrary-ord"

// Spec registers the rule with the argument contract of toolbelt/lint-arbitrary-ord.sh: advisory WARN
// rows, exit 0 (3 on a usage or env error).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-arbitrary-ord.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the partial list (env line, rows, exit 3)
		// and word an unreadable file "<lint>: cannot scan <path>: <reason>".
		ContinueOnWalkError: true,
		ShortScanError:      true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

var (
	call        = regexp.MustCompile(`BOrd\.make\([[:space:]]*[a-z][A-Za-z0-9_]*[[:space:]]*\)`)
	withReason  = regexp.MustCompile(`lint-arbitrary-ord:[[:space:]]*reviewed[[:space:]]+[^[:space:]]`)
	marker      = regexp.MustCompile(`lint-arbitrary-ord:[[:space:]]*reviewed`)
	commentLine = regexp.MustCompile(`^[[:space:]]*//`)
)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports each line whose code (linescan.Code: no comment, Javadoc or literal text; truth fix T1,
// the awk matched the raw text before the // comment) calls BOrd.make on a lowercase identifier: "BOrd.make
// from a variable", or "reviewed marker without a reason" when the line's // comment (javasrc.LineCommentAt),
// or else a // comment line right above, carries a bare marker. A marker with a reason suppresses the row.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	code := linescan.Code(f)
	prev := ""
	for i, ln := range f.Raw {
		comment := ""
		if at := javasrc.LineCommentAt(ln); at >= 0 {
			comment = ln[at:]
		}
		if call.MatchString(code[i]) {
			st := state(comment)
			if st == "" && commentLine.MatchString(prev) {
				st = state(prev)
			}
			switch st {
			case "bare":
				out = append(out, lint.Finding{Line: i + 1, Msg: "reviewed marker without a reason (add one): " + linescan.Source(ln)})
			case "":
				out = append(out, lint.Finding{Line: i + 1, Msg: "BOrd.make from a variable: " + linescan.Source(ln)})
			}
		}
		prev = ln
	}
	return out, nil
}

// state is the awk marker_state: "reason", "bare" (a marker with nothing after it) or "" (no marker).
func state(text string) string {
	switch {
	case withReason.MatchString(text):
		return "reason"
	case marker.MatchString(text):
		return "bare"
	}
	return ""
}
