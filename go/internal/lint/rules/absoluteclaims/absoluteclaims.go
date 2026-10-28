// Package absoluteclaims is lint-absolute-claims (G38): a report of the kit's absolute claims ("never", "always",
// "silently", "forbidden", "throws") that cite only corpus inference or retros. METHODOLOGY K33: such a rule is
// checked against Tridium-authored training and code before it is kept, because the course reversed four kit rules
// generalised from one code path. The contract is the deferred G38 spec
// (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-process.md, C51); toolbelt/lint-absolute-claims.sh is the wrapper
// and its header is the user-facing contract.
//
// It is a lint.TreeRule over each root (normally build-n4-module-kit/types): every *.md. The evidence of a line is
// its claim unit: the Markdown bullet or paragraph that holds it (a bullet runs to the next bullet, heading or blank
// line). A unit is strong when one of its [ev: ...] tags starts with code, course, devguide, CERT or javap, or its
// text holds a `.java:<n>` line cite. A row is each line of a unit that is not strong holding one of the words
// (whole word, any case), naming the first one, the line and the unit's tags. Rows are INFO: the report has no
// verdict and never fails.
package absoluteclaims

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column.
const ID = "lint-absolute-claims"

// Spec registers the rule: [--strict] [--] <types-dir>... (each root on its own), INFO rows (exit 0 even under
// --strict). Its roots are kit doc directories, not source roots, so n4lint --all leaves it out (ModuleRoot).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-absolute-claims.sh [--strict] <types-dir>...",
		Select:     lint.Selection{Names: []string{"*.md"}},
		EmptyError: lint.EmptyRow{Msg: "no *.md under the root (wrong path: pass build-n4-module-kit/types)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Info }

var (
	word     = regexp.MustCompile(`(?i)\b(never|always|silently|forbidden|throws)\b`)
	tag      = regexp.MustCompile(`\[ev: [^\]]*\]`)
	strong   = regexp.MustCompile(`^\[ev: (code|course|devguide|CERT|javap)\b`)
	javaLine = regexp.MustCompile(`\.java:[0-9]`)
	bullet   = regexp.MustCompile(`^\s*([-*+]|[0-9]+\.)\s`)
)

// CheckTree reports every file in walk order.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for _, f := range files {
		out = append(out, report(f)...)
	}
	return out, nil
}

// report lists the absolute-claim lines of one document whose claim unit is not strong.
func report(f *javasrc.File) []lint.Finding {
	var out []lint.Finding
	lines := f.Raw
	for start := 0; start < len(lines); {
		end := unitEnd(lines, start)
		unit := strings.Join(lines[start:end], "\n")
		tags := tag.FindAllString(unit, -1)
		if !isStrong(unit, tags) {
			shown := "-"
			if len(tags) > 0 {
				shown = strings.Join(tags, " ")
			}
			for i := start; i < end; i++ {
				if w := word.FindString(lines[i]); w != "" {
					claim := strings.TrimSpace(strings.TrimRight(lines[i], "\r"))
					if r := []rune(claim); len(r) > 120 {
						claim = string(r[:117]) + "..."
					}
					out = append(out, lint.Finding{Path: f.Path, Line: i + 1, Msg: strings.ToLower(w) + ": " + claim + "  " + shown})
				}
			}
		}
		start = end
	}
	return out
}

// unitEnd is the index after the claim unit that starts at lines[i]: one line for a blank line or a heading, else
// the lines up to the next blank line, heading or bullet.
func unitEnd(lines []string, i int) int {
	if t := strings.TrimSpace(lines[i]); t == "" || strings.HasPrefix(t, "#") {
		return i + 1
	}
	j := i + 1
	for j < len(lines) {
		t := strings.TrimSpace(lines[j])
		if t == "" || strings.HasPrefix(t, "#") || bullet.MatchString(lines[j]) {
			break
		}
		j++
	}
	return j
}

func isStrong(unit string, tags []string) bool {
	if javaLine.MatchString(unit) {
		return true
	}
	for _, t := range tags {
		if strong.MatchString(t) {
			return true
		}
	}
	return false
}
