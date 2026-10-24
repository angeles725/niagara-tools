// Package manuallabels is lint-manual-labels: every quoted UI label of an operator manual ("…", “…”, «…»)
// must exist in the UI source of its <ui-dir>s (advisory WARN rows and one SUMMARY total). Port of
// toolbelt/lint-manual-labels.sh (a python3 heredoc); the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins. Python's text semantics (errors="replace" reads,
// html.unescape, re's \s and \w, NFKD, casefold) come from internal/htmltext.
//
// It is a Joined lint.TreeRule (one haystack over every <ui-dir>, then the manual) and a lint.Configurer
// that reads the manual (Mode FileFirst) before the walk.
package manuallabels

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// ID is the lint name printed in every row.
const ID = "lint-manual-labels"

// exts are the UI source files of the haystack, matched in any case as Python's f.lower().endswith(EXTS).
var exts = []string{"html", "htm", "js", "mjs", "cjs", "json", "lexicon", "properties", "java", "xml"}

// Spec registers the rule with the argument contract of toolbelt/lint-manual-labels.sh:
// [--strict] <manual> <ui-dir>..., every ui-dir walked (dot, node_modules and *-worktrees directories pruned).
func Spec() lint.Spec {
	names := make([]string, len(exts))
	for i, e := range exts {
		var b strings.Builder
		b.WriteString("*.")
		for _, c := range e {
			fmt.Fprintf(&b, "[%c%c]", c, c-'a'+'A')
		}
		names[i] = b.String()
	}
	return lint.Spec{
		Tree:   &rule{},
		Mode:   lint.FileFirst,
		Option: "manual",
		Joined: true,
		Usage:  "usage: lint-manual-labels.sh [--strict] <manual.html|manual.md> <ui-dir> [<ui-dir>...]",
		Select: lint.Selection{Names: names, Prune: []string{"node_modules"}},
		// a ui-dir with no UI file is a wrong path: every label reported missing (exit 0) hid it (truth fix 2026-10-24)
		EmptyError: lint.EmptyRow{Msg: "no UI source files found (*." + strings.Join(exts, " *.") + "; wrong path or empty source tree)"},
	}
}

// rule holds the manual: its path as given and its text as the labels are read from it.
type rule struct{ manual, text string }

func (*rule) ID() string              { return ID }
func (*rule) Severity() lint.Severity { return lint.Warn }

// Configure reads the manual. An HTML manual (.html/.htm in any case) is read without its script, style and
// code elements, comments and tags (line numbers kept), then html.unescape'd (a pre element without code
// stays text, as a bare Markdown fence: a diagram); a Markdown manual (.md, .markdown) without its
// language-tagged fenced blocks and code spans (markdownCode); any other manual as is.
func (r *rule) Configure(a lint.Args) error {
	data, err := os.ReadFile(a.Option)
	if err != nil {
		return fmt.Errorf("%s: cannot read manual %s: %s\n", ID, a.Option, strerror.Text(err))
	}
	r.manual, r.text = a.Option, htmltext.Decode(data)
	switch low := strings.ToLower(a.Option); {
	case strings.HasSuffix(low, ".html") || strings.HasSuffix(low, ".htm"):
		r.text = htmltext.Unescape(htmltext.DropMarkup(r.text, "code"))
	case strings.HasSuffix(low, ".md") || strings.HasSuffix(low, ".markdown"):
		r.text = markdownCode(r.text)
	}
	return nil
}

// CheckTree builds the haystack of every UI file under every <ui-dir>, then checks each label of the manual.
func (r *rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	hay, stems := map[string]bool{}, map[string]bool{}
	for _, f := range files {
		us, ss := units(htmltext.Decode([]byte(strings.Join(f.Raw, "\n"))))
		for _, u := range us {
			hay[norm(u)] = true
		}
		for _, u := range ss {
			if n := norm(u); hasLetter(n) {
				stems[n] = true
			}
		}
	}
	sorted := make([]string, 0, len(hay))
	for u := range hay {
		sorted = append(sorted, u)
	}
	sort.Strings(sorted)
	var out []lint.Finding
	checked, missing := 0, 0
	for _, l := range labels(r.text) {
		checked++
		if n := norm(l.text); !found(n, sorted) && !templated(n, stems) {
			missing++
			out = append(out, lint.Finding{Path: r.manual, Line: l.line, Msg: `label "` + l.text + `" not found in the UI source`})
		}
	}
	return append(out, lint.Finding{Severity: lint.Summary, Msg: fmt.Sprintf("checked=%d missing=%d", checked, missing)}), nil
}

// norm is how a label and a UI text unit compare: accents dropped (NFKD without combining marks), whitespace
// collapsed and stripped, a trailing ':' dropped, case folded.
func norm(s string) string {
	s = htmltext.Strip(htmltext.CollapseSpace(htmltext.StripMarks(s)))
	return htmltext.CaseFold(htmltext.Strip(strings.TrimRight(s, ":")))
}

// templated reports whether a normalized label is a stem the UI joins a runtime value to, at a word boundary,
// plus one word: "Guardado (N)" for "Guardado (" + n + ")", "HTTP 400" for "HTTP " + status (truth fix
// 2026-10-24: the manual quotes the label as the operator reads it, with a placeholder or a sample value).
func templated(label string, stems map[string]bool) bool {
	for i := range label {
		if i == 0 || (htmltext.IsAlnum(firstRune(label[i:])) && htmltext.IsAlnum(lastRune(label[:i]))) {
			continue
		}
		if rest := strings.TrimLeft(label[i:], " :"); stems[label[:i]] && rest != "" && !strings.Contains(rest, " ") {
			return true
		}
	}
	return false
}

// found reports whether a normalized label equals a unit or is its prefix ending at a word boundary (the
// unit's next code point is not str.isalnum()); units is sorted, so the units it prefixes are contiguous.
func found(label string, units []string) bool {
	for i := sort.SearchStrings(units, label); i < len(units) && strings.HasPrefix(units[i], label); i++ {
		if rest := units[i][len(label):]; rest == "" || !htmltext.IsAlnum(firstRune(rest)) {
			return true
		}
	}
	return false
}
