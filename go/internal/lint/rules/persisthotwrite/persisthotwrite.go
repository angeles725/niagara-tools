// Package persisthotwrite is the lint-persist-hot-write rule: a non-transient slot-o-matic property's
// setter called from changed(Property ...) or from any zero-arg method changed() calls (one hop), with no
// cadence guard (a Clock.millis( or a `% n ==/!=` test) on that line or the 5 above it in the same body.
// Port of the awk in toolbelt/lint-persist-hot-write.sh (the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins); it shares changed(), bodies, callees and properties
// with lint-changed-hot-write through rules/hotwrite.
package persisthotwrite

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/hotwrite"
)

// ID is the lint name printed in every row.
const ID = "lint-persist-hot-write"

// Spec registers the rule with the argument contract of toolbelt/lint-persist-hot-write.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Tree:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-persist-hot-write.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	setterDecl = regexp.MustCompile(`(void|public|private|protected|static)[[:space:]]+[A-Za-z_<>\[\],. ]*set[A-Z]`)
	modulo     = regexp.MustCompile(`%[[:space:]]*[A-Za-z0-9_]+[[:space:]]*(==|!=)`)
	clockLocal = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*=[[:space:]]*(\([A-Za-z]+\)[[:space:]]*)?Clock\.millis\(`)
	comparison = regexp.MustCompile(`<=|>=|==|!=|[^-]>|<`)
	ifHead     = regexp.MustCompile(`(^|[^A-Za-z0-9_])(if|while)[[:space:]]*\(`)
)

// CheckTree reads every file under the root once for the fields it accumulates (hotwrite.Accumulators), then
// reports each file in walk order (check).
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	acc := hotwrite.Accumulators(files)
	var out []lint.Finding
	for _, f := range files {
		out = append(out, check(f, acc)...)
	}
	return out, nil
}

// check WARNs every setter call (hotwrite.SetterCall) of a non-transient property in changed() or a one-hop
// callee that has no cadence guard within 5 lines and can mark the station dirty on every cycle
// (hotwrite.Hot: not under an on-change if, and not a simple value that is not computed fresh), property by
// property, body by body (changed() first), line by line.
func check(f *javasrc.File, acc map[string]bool) []lint.Finding {
	lines := hotwrite.View(f)
	changed, _, ok := hotwrite.ChangedBody(f, lines)
	if !ok {
		return nil
	}
	bodies := []hotwrite.Body{changed}
	for _, name := range hotwrite.Callees(lines, changed.Start, changed.End) {
		if b, ok := hotwrite.ZeroArg(f, lines, name); ok {
			bodies = append(bodies, b)
		}
	}
	var out []lint.Finding
	for _, prop := range hotwrite.Properties(lines) {
		if prop.Transient {
			continue
		}
		setter := hotwrite.Setter(prop.Name) + "("
		for _, b := range bodies {
			for _, i := range hotwrite.HotCalls(lines, b, prop, acc) {
				at := hotwrite.SetterCall(prop.Name).FindStringIndex(lines[i])[0]
				if setterDecl.MatchString(lines[i]) || guarded(lines, b.Start, i, at) {
					continue
				}
				out = append(out, lint.Finding{Path: f.Path, Line: i + 1, Msg: setter + "...) writes non-transient slot '" +
					prop.Name + "' reached from changed(), no cadence guard (Clock.millis()/counter) within 5 lines --" +
					" integrate in memory, checkpoint on a periodic tick + in stopped(), seed once on start"})
			}
		}
	}
	return out
}

// guarded reports a cadence guard for the setter call at lines[i], byte at, within the 5 lines above it (not
// above the body's first line, start): a `% n ==/!=` counter test, or a Clock.millis() read in a condition
// that gates the write (a comparison or an if, reading Clock.millis() itself or a local assigned from it).
// On the call's own line only the text before the call counts and it must be an if, so a clock read inside
// the written value (`setX((double) Clock.millis())`) never guards it.
func guarded(lines []string, start, i, at int) bool {
	var locals []string
	for j := start; j <= i; j++ {
		for _, m := range clockLocal.FindAllStringSubmatch(lines[j], -1) {
			locals = append(locals, m[1])
		}
	}
	reads := func(t string) bool {
		if strings.Contains(t, "Clock.millis(") {
			return true
		}
		for _, v := range locals {
			if regexp.MustCompile(`(^|[^A-Za-z0-9_])` + v + `([^A-Za-z0-9_]|$)`).MatchString(t) {
				return true
			}
		}
		return false
	}
	for j := i; j >= start && j >= i-5; j-- {
		t := lines[j]
		if j == i {
			t = t[:at]
			if modulo.MatchString(t) || ifHead.MatchString(t) && reads(t) {
				return true
			}
			continue
		}
		if modulo.MatchString(t) || reads(t) && (comparison.MatchString(t) || ifHead.MatchString(t)) {
			return true
		}
	}
	return false
}
