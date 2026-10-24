// Package wsplacement is the lint-ws-placement rule: a method that adds a freshly created component to a
// parent (an add( call whose second argument is a creation expression or a local assigned from one) and
// never mentions wsAnnotation, BWsAnnotation or WsGrid, so Workbench staggers the child onto its siblings.
// Port of the awk in toolbelt/lint-ws-placement.sh (the wrapper keeps the user-facing contract and its
// header documents the rule and its mutation pins). Code is the javasrc Strip view (comments blanked,
// literal text kept) cut into javasrc Methods; matching keeps gawk's POSIX leftmost-longest semantics.
package wsplacement

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-ws-placement"

// Spec registers the rule with the argument contract of toolbelt/lint-ws-placement.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Each,
		Usage:      "usage: lint-ws-placement.sh [--strict] <src-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	guard      = rx(`wsAnnotation|BWsAnnotation|WsGrid`)
	assign     = rx(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[^=][^;]*`)
	ident      = rx(`^[A-Za-z_][A-Za-z0-9_]*`)
	assignHead = rx(`^[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*`)
	addCall    = rx(`(^|[^A-Za-z0-9_])add[[:space:]]*\(`)
	numeric    = rx(`^-?[0-9]+[lL]?$`)
	castNext   = rx(`^\([A-Za-z_][A-Za-z0-9_.<>]*\)[[:space:]]*[^[:space:]]`)
	castHead   = rx(`^\([A-Za-z_][A-Za-z0-9_.<>]*\)[[:space:]]*`)
	newType    = rx(`^new[[:space:]]+B[A-Z][A-Za-z0-9_]*`)
	newHead    = rx(`^new[[:space:]]+`)
	// value types that never get a wire-sheet glyph
	valueType = rx(`^(BStatus[A-Za-z0-9_]*|BWsAnnotation|BFacets|BOrd|BLink|BConversionLink|BAbsTime|BRelTime|BString|BDouble|BFloat|BInteger|BLong|BBoolean|BEnumRange|BFormat|BIcon|BNameMap)$`)
	factory   = rx(`^([A-Za-z_][A-Za-z0-9_.]*\.)?(create|newInstance|newCopy)[[:space:]]*\(`)
	openParen = rx(`[[:space:]]*\($`)
)

const advice = " never sets a wsAnnotation -- the wire sheet staggers it onto its siblings; add a grid position" +
	" if absent with Flags.READONLY, never re-place one (types/logic-authoring.md §Wire-sheet placement)"

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check WARNs once per add( line of an unguarded method whose second argument is a new component.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	var out []lint.Finding
	for _, m := range f.Methods() {
		lines := code[m.Start-1 : m.End]
		if guarded(lines) {
			continue
		}
		made := locals(lines)
		for k, c := range lines {
			joined := strings.Join(lines[k:min(k+6, len(lines))], " ")
			if a1, what, ok := firstAdd(c, joined, made); ok {
				out = append(out, lint.Finding{Line: m.Start + k, Msg: "add(" + a1 + ", " + what + ") in " + m.Name + "()" + advice})
			}
		}
	}
	return out, nil
}

// guarded reports whether a method line mentions one of the placement words.
func guarded(lines []string) bool {
	for _, c := range lines {
		if guard.MatchString(c) {
			return true
		}
	}
	return false
}

// locals maps every `<name> = [casts] <creation>` of the method to what it creates.
func locals(lines []string) map[string]string {
	made := map[string]string{}
	for _, t := range lines {
		for {
			loc := assign.FindStringIndex(t)
			if loc == nil {
				break
			}
			s := t[loc[0]:loc[1]]
			t = t[loc[1]:]
			if w := creates(stripCasts(s[len(assignHead.FindString(s)):])); w != "" {
				made[ident.FindString(s)] = w
			}
		}
	}
	return made
}

// firstAdd reads the add( calls of line c (arguments from joined, c and the next lines) and returns the
// first one adding a new component: its trimmed first argument and what the second creates.
func firstAdd(c, joined string, made map[string]string) (a1, what string, ok bool) {
	for pos := 0; ; {
		loc := addCall.FindStringIndex(c[pos:])
		if loc == nil {
			return "", "", false
		}
		pos += loc[1]
		args, closed := splitArgs(joined, pos)
		if !closed || len(args) < 2 || numeric.MatchString(linescan.Trimmed(args[0])) {
			continue
		}
		a2 := stripCasts(args[1])
		what = creates(a2)
		if w, ok := made[a2]; what == "" && ok {
			what = a2 + " = " + w
		}
		if what != "" {
			return linescan.Trimmed(args[0]), what, true
		}
	}
}

// splitArgs reads the argument list that opens just before s[from]: its top-level arguments (untrimmed),
// split at depth-0 commas outside "..." and '...' (\ escapes honoured), and whether its ")" was found.
func splitArgs(s string, from int) (args []string, closed bool) {
	depth, quote := 0, byte(0)
	var cur strings.Builder
	for j := from; j < len(s); j++ {
		c := s[j]
		switch {
		case quote != 0:
			cur.WriteByte(c)
			if c == '\\' && j+1 < len(s) {
				j++
				cur.WriteByte(s[j])
			} else if c == quote {
				quote = 0
			}
			continue
		case c == '"' || c == '\'':
			quote = c
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			if depth == 0 {
				if linescan.Trimmed(cur.String()) != "" || len(args) > 0 {
					args = append(args, cur.String())
				}
				return args, true
			}
			depth--
		case c == ',' && depth == 0:
			args = append(args, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteByte(c)
	}
	return args, false
}

// stripCasts trims a and drops the leading `(Type)` casts that something follows.
func stripCasts(a string) string {
	a = linescan.Trimmed(a)
	for castNext.MatchString(a) {
		a = a[len(castHead.FindString(a)):]
	}
	return a
}

// creates is what a creation expression at the start of a creates, or "" when it is not one.
func creates(a string) string {
	if t := newType.FindString(a); t != "" {
		t = t[len(newHead.FindString(t)):]
		if valueType.MatchString(t) {
			return ""
		}
		return "new " + t + "()"
	}
	if t := factory.FindString(a); t != "" {
		return openParen.ReplaceAllString(t, "") + "(...)"
	}
	return ""
}
