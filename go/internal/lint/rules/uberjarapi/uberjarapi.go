// Package uberjarapi is the lint-uberjar-api-conflict rule: a library both bundled with uberjar("g:a:v") and
// declared with api(...) or implementation(...) in one *.gradle.kts puts two copies of its classes in two
// classloader tiers. Port of toolbelt/lint-uberjar-api-conflict.sh (the wrapper keeps the user-facing
// contract; its header documents the rule and its mutation pins).
package uberjarapi

import (
	"regexp"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-uberjar-api-conflict"

// Spec registers the rule with the argument contract of toolbelt/lint-uberjar-api-conflict.sh: advisory
// WARN rows, exit 0 (3 on a usage or env error).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:        rule{},
		Mode:        lint.Raw,
		Usage:       "usage: lint-uberjar-api-conflict.sh <src-root>",
		Select:      lint.Selection{Names: []string{"*.gradle.kts"}},
		BarePathRow: true,
		// Reason: bash used awk_lint_list (the partial list is scanned: env line, rows, exit 3) with
		// AL_EMPTY_OK: report-module runs it on every module root, and one with no *.gradle.kts has nothing
		// to check (tests/lint-empty-root.bats ER5).
		ContinueOnWalkError: true,
		EmptyOK:             true,
		ShortScanError:      true,
	}
}

// dep is the bash grep -Eo of one configuration call: <config>("group:artifact[:version]"), either quote.
func dep(config string) *regexp.Regexp {
	return regexp.MustCompile(config + `\(["']([A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+)(:[^"']*)?["']\)`)
}

var uberjar, api, implementation = dep("uberjar"), dep("api"), dep("implementation")

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports, in byte order, every group:artifact the file both uberjar()s and api()s or
// implementation()s.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	bundled, declared := coords(f.Raw, uberjar), coords(f.Raw, api, implementation)
	var both []string
	for ga := range bundled {
		if declared[ga] {
			both = append(both, ga)
		}
	}
	sort.Strings(both)
	out := make([]lint.Finding, 0, len(both))
	for _, ga := range both {
		out = append(out, lint.Finding{Msg: ga + " is both uberjar()d and api()d"})
	}
	return out, nil
}

// coords are the group:artifact coordinates the configuration calls name, on any line.
func coords(lines []string, configs ...*regexp.Regexp) map[string]bool {
	set := map[string]bool{}
	for _, ln := range lines {
		for _, re := range configs {
			for _, m := range re.FindAllStringSubmatch(ln, -1) {
				set[m[1]] = true
			}
		}
	}
	return set
}
