// Package extordvalue is the lint-wb-external-ord-value rule: a -wb method that resolves an ORD, reads a
// live value/status and never leases a refresh with registerForComponentEvents(. Port of the awk in
// toolbelt/lint-wb-external-ord-value.sh (the wrapper keeps the user-facing contract and its header documents
// the rule and its mutation pins).
package extordvalue

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-wb-external-ord-value"

// Spec registers the rule with the argument contract of toolbelt/lint-wb-external-ord-value.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-wb-external-ord-value.sh [--strict] <wb-src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

// resolve is an ORD resolve: `<ident>Ord().get(`, or `<ident>Ord.get(` / a bare `ord.get(` as a whole
// identifier (never `record.get(`).
var resolve = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*Ord\(\)[[:space:]]*\.get\(|(^|[^A-Za-z0-9_])([A-Za-z_][A-Za-z0-9_]*Ord|ord)[[:space:]]*\.get\(`)

const msg = "() resolves an external ORD and reads its value/status with no registerForComponentEvents(target,0) lease in scope -- subject-subtree depth does not cover it; add loadSlots() + registerForComponentEvents(target,0) for the resolved target"

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports, per method, its first ORD-resolve line when the method also reads a live value/status
// and calls registerForComponentEvents( nowhere.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	var out []lint.Finding
	for _, m := range f.Methods() {
		body := code[m.Start-1 : m.End]
		at, live, leased := 0, false, false
		for i, ln := range body {
			if at == 0 && resolve.MatchString(ln) {
				at = m.Start + i
			}
			live = live || strings.Contains(ln, `.get("out")`) || strings.Contains(ln, "getOutStatusValue(") || strings.Contains(ln, "BStatusValue")
			leased = leased || strings.Contains(ln, "registerForComponentEvents(")
		}
		if at != 0 && live && !leased {
			out = append(out, lint.Finding{Line: at, Msg: m.Name + msg})
		}
	}
	return out, nil
}
