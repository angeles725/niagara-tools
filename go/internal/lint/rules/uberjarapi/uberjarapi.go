// Package uberjarapi is the lint-uberjar-api-conflict rule: a library both bundled with uberjar("g:a:v") and
// declared with api(...) or implementation(...) in one *.gradle.kts puts two copies of its classes in two
// classloader tiers. Port of toolbelt/lint-uberjar-api-conflict.sh (the wrapper keeps the user-facing
// contract; its header documents the rule and its mutation pins).
package uberjarapi

import (
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/ktssrc"
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

// coordinate is a dependency string's group:artifact, its version (which may be a template) dropped.
var coordinate = regexp.MustCompile(`^([A-Za-z0-9_.-]+:[A-Za-z0-9_.-]+)(:.*)?$`)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check reports, in byte order, every group:artifact the file both uberjar()s and api()s or
// implementation()s. The calls are read as Kotlin tokens (truth fix T1; the bash grep read raw text): a
// configuration name, "(", one string literal, ")", with any layout between them; comments and strings
// hold no call. A script ktssrc cannot read is a scan error (exit 3).
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks, err := ktssrc.Tokens(strings.Join(f.Raw, "\n"))
	if err != nil {
		return nil, err
	}
	bundled, declared := map[string]bool{}, map[string]bool{}
	for k := 0; k+3 < len(toks); k++ {
		name, open, dep, shut := toks[k], toks[k+1], toks[k+2], toks[k+3]
		if name.Kind != ktssrc.Ident || open.Text != "(" || dep.Kind != ktssrc.String || shut.Text != ")" {
			continue
		}
		m := coordinate.FindStringSubmatch(dep.Value)
		switch {
		case m == nil:
		case name.Text == "uberjar":
			bundled[m[1]] = true
		case name.Text == "api" || name.Text == "implementation":
			declared[m[1]] = true
		}
	}
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
