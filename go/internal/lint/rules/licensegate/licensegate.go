// Package licensegate is lint-license-isoperational-gate: a LICENSED class (getLicenseFeature() returns a
// feature) whose changed() / do<Upper>… action / servlet write callback acts with no isOperational() /
// isFault() / isFatalFault() gate opening it. Port of the awk in toolbelt/lint-license-isoperational-gate.sh
// (the wrapper keeps the user-facing contract and its header documents the rule and its mutation pins).
//
// It is a per-file lint.Rule: a file's rows depend on that file alone. The awk read mb_strip's lines
// (javasrc.Stripped: comments blanked, literal text kept) and mb_parse's methods (javasrc.Methods); regex
// matching keeps gawk's POSIX leftmost-longest semantics (regexp Longest) under LC_ALL=C.
package licensegate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-license-isoperational-gate"

// Spec registers the rule with the argument contract of toolbelt/lint-license-isoperational-gate.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.One,
		Usage:  "usage: lint-license-isoperational-gate.sh [--strict] <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	abstractDecl = rx(`getLicenseFeature[[:space:]]*\([[:space:]]*\)[[:space:]]*;`)
	licenseDecl  = rx(`Feature[[:space:]]+getLicenseFeature[[:space:]]*\([[:space:]]*\)`)
	space        = rx(`[[:space:]]`)
	action       = rx(`^do[A-Z]`)
	readHandler  = rx(`^do(Get|Head|Options|Trace)$`)
	superCall    = rx(`super\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*\([^;]*\)[[:space:]]*;`)
	leadSpace    = rx(`^[[:space:]]+`)
	gate         = rx(`(^|[^A-Za-z0-9_.])(this\.)?(isOperational|isFault|isFatalFault)[[:space:]]*\(`)
	bareReturn   = rx(`return[[:space:]]*;`)
)

const msg = "licensed class: %s() acts with no isOperational()/isFault()/isFatalFault() gate -- the license fault does not stop callbacks; add `if (!isOperational()) return;` (types/security.md §5.1)"

// Check is the awk's END block over the file's stripped lines.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	if !licensed(code) {
		return nil, nil
	}
	var out []lint.Finding
	for _, m := range f.Methods() {
		if m.Name != "changed" && !action.MatchString(m.Name) || readHandler.MatchString(m.Name) {
			continue
		}
		t := superCall.ReplaceAllString(blockBody(code, m.Start-1), "")
		head := leadSpace.ReplaceAllString(t, "")
		if k := strings.IndexAny(head, "{;"); k >= 0 {
			head = head[:k+1]
		}
		if gate.MatchString(head) {
			continue
		}
		if space.ReplaceAllString(bareReturn.ReplaceAllString(t, ""), "") == "" {
			continue
		}
		out = append(out, lint.Finding{Line: m.Start, Msg: fmt.Sprintf(msg, m.Name)})
	}
	return out, nil
}

// licensed reports a getLicenseFeature() declaration whose body is neither empty nor `return null;`.
func licensed(code []string) bool {
	lic := false
	for i, ln := range code {
		if abstractDecl.MatchString(ln) || !licenseDecl.MatchString(ln) {
			continue
		}
		if b := space.ReplaceAllString(blockBody(code, i), ""); b != "" && b != "returnnull;" {
			lic = true
		}
	}
	return lic
}

// blockBody is the awk block_body(): the text of the brace block that opens at or after 0-based line from,
// from the first "{" to its matching "}" (exclusive), a "\n" after each line it spans. Like the awk it counts
// every brace, inside literals too.
func blockBody(code []string, from int) string {
	var b strings.Builder
	depth, started := 0, false
	for i := from; i < len(code); i++ {
		for j := 0; j < len(code[i]); j++ {
			switch ch := code[i][j]; {
			case ch == '{':
				depth++
				if !started {
					started = true
					continue
				}
			case ch == '}':
				depth--
				if started && depth == 0 {
					return b.String()
				}
			}
			if started {
				b.WriteByte(code[i][j])
			}
		}
		if started {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
