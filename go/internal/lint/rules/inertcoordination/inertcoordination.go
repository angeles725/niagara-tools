// Package inertcoordination is the lint-inert-coordination rule: a class whose units() method returns a
// structurally singleton list (singletonList(, or a one-argument Arrays.asList() while it still declares
// queue/token/stagger coordination state WARNs at the units() declaration: the coordination machinery is
// dead code after a per-unit move (PANCCADIA BDefrostController, 2026-09-23). Port of the awk in
// toolbelt/lint-inert-coordination.sh (the wrapper keeps the user-facing contract and its header documents
// the rule, its limitations and its mutation pins).
package inertcoordination

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-inert-coordination"

// Spec registers the rule with the argument contract of toolbelt/lint-inert-coordination.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-inert-coordination.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	unitsDecl = rx(`[A-Za-z_][A-Za-z0-9_<>\[\], ]*[[:space:]]units\(\)`)
	queueType = rx(`(Deque|Queue)[[:space:]]*<`)
	assigned  = rx(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=`)
	coordName = rx(`[^A-Za-z0-9_]([a-z][A-Za-z0-9_]*(Token|Stagger)|token|stagger|Token|Stagger)[A-Za-z0-9_]*`)
)

// Check WARNs once at the units() declaration when its body is singleton-shaped and the file declares
// coordination state.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := make([]string, len(f.Raw)) // each line cut at its first "//", as the awk's sub(/\/\/.*$/, "")
	for i, ln := range f.Raw {
		lines[i], _, _ = strings.Cut(ln, "//")
	}
	decl, ok := declaration(lines)
	if !ok {
		return nil, nil
	}
	end, started := bodyEnd(lines, decl)
	if !started || !singleton(lines[decl:end+1]) {
		return nil, nil
	}
	found := coordination(lines)
	if len(found) == 0 {
		return nil, nil
	}
	return []lint.Finding{{Line: decl + 1, Msg: "units() returns a structurally singleton list while " +
		strings.Join(found, ", ") + " (queue/token/stagger state) are still declared -- the coordination machinery" +
		" is dead code after the per-unit move; coordinate at the parent that aggregates multiple units, or drop" +
		" the unused fields"}}, nil
}

// declaration is the first line whose `<type> units()` is followed by nothing, "{" or "throws": a
// declaration, not a call such as `return units().size();`.
func declaration(lines []string) (int, bool) {
	for i, ln := range lines {
		if loc := unitsDecl.FindStringIndex(ln); loc != nil {
			t := strings.TrimLeft(ln[loc[1]:], " \t\n\v\f\r")
			if t == "" || strings.HasPrefix(t, "{") || strings.HasPrefix(t, "throws") {
				return i, true
			}
		}
	}
	return 0, false
}

// bodyEnd is the line where the braces counted from line start return to depth 0 after one opened (the last
// line when they never do) and whether one opened.
func bodyEnd(lines []string, start int) (int, bool) {
	depth, started := 0, false
	for i := start; i < len(lines); i++ {
		for _, c := range []byte(lines[i]) {
			switch c {
			case '{':
				depth, started = depth+1, true
			case '}':
				depth--
			}
		}
		if started && depth == 0 {
			return i, true
		}
	}
	return len(lines) - 1, started
}

// singleton reports a singletonList( on a body line, or an Arrays.asList( with no "," after it on its line.
func singleton(body []string) bool {
	for _, ln := range body {
		if strings.Contains(ln, "singletonList(") {
			return true
		}
		if _, tail, ok := strings.Cut(ln, "Arrays.asList("); ok && !strings.Contains(tail, ",") {
			return true
		}
	}
	return false
}

// coordination are the coordination-state names of the file in first-seen order: per line, the first
// `<name> =` of a line typed Deque<…>/Queue<…> (or "(Deque/Queue field)"; skipped when it is a substring of
// the names so far, as the awk's index() test did), then every token/stagger identifier, each once.
func coordination(lines []string) []string {
	var found []string
	for _, ln := range lines {
		if queueType.MatchString(ln) {
			name := "(Deque/Queue field)"
			if m := assigned.FindString(ln); m != "" {
				name = strings.TrimRight(m[:len(m)-1], " \t\n\v\f\r")
			}
			if !strings.Contains(strings.Join(found, ", "), name) {
				found = append(found, name)
			}
		}
		for _, m := range coordName.FindAllString(" "+ln, -1) {
			if seg := m[1:]; !contains(found, seg) {
				found = append(found, seg)
			}
		}
	}
	return found
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
