// Package changedhotwrite is the lint-changed-hot-write rule: a changed(Property p, ...) callback with six or
// more `p ==` branches that calls execute() with no Clock.millis()/Clock.schedule rate guard in its body,
// where changed() or its zero-arg execute() writes a non-transient slot-o-matic property, WARNs once at the
// changed() line. Port of the awk in toolbelt/lint-changed-hot-write.sh (the wrapper keeps the user-facing
// contract and its header documents the rule and its mutation pins); it shares changed(), bodies and
// properties with lint-persist-hot-write through rules/hotwrite.
package changedhotwrite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/hotwrite"
)

// ID is the lint name printed in every row.
const ID = "lint-changed-hot-write"

// minBranches is how many `p ==` branches make a changed() hot.
const minBranches = 6

// Spec registers the rule with the argument contract of toolbelt/lint-changed-hot-write.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-changed-hot-write.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// text is lines[from..to], each prefixed by "\n" (the awk's body string).
func text(lines []string, from, to int) string {
	var b strings.Builder
	for i := from; i <= to; i++ {
		b.WriteString("\n" + lines[i])
	}
	return b.String()
}

func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := hotwrite.View(f)
	body, prop, ok := hotwrite.ChangedBody(f, lines)
	if !ok {
		return nil, nil
	}
	changed := text(lines, body.Start, body.End)
	n := len(regexp.MustCompile(`[^A-Za-z0-9_]`+regexp.QuoteMeta(prop)+`[[:space:]]*==`).FindAllStringIndex(changed, -1))
	if n < minBranches || !strings.Contains(changed, "execute()") ||
		strings.Contains(changed, "Clock.millis(") || strings.Contains(changed, "Clock.schedule") {
		return nil, nil
	}
	exec := changed
	if b, ok := hotwrite.ZeroArg(f, lines, "execute"); ok {
		exec = text(lines, b.Start, b.End)
	}
	for _, slot := range hotwrite.NonTransient(lines) {
		setter := hotwrite.SetterCall(slot)
		if setter.MatchString(exec) || setter.MatchString(changed) {
			return []lint.Finding{{Line: body.Start + 1, Msg: fmt.Sprintf("changed(%s) has %d '%s ==' branches guarding execute(),"+
				" no Clock.millis()/schedule rate guard, writes non-transient slot '%s' -- debounce the callback"+
				" (leading-edge Clock.millis() gate) or move the periodic write to a tick", prop, n, prop, slot)}}, nil
		}
	}
	return nil, nil
}
