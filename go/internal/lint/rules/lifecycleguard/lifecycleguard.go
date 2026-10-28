// Package lifecycleguard is lint-lifecycle-guard: component lifecycle callbacks that work when the component is not
// running (corpus B1242 §1242.7 K4, B1416 §1416.3). toolbelt/lint-lifecycle-guard.sh is the wrapper and its header
// is the user-facing contract.
//
// A per-file rule over *.java, read as javasrc tokens (comments blanked, a literal one token). Rows:
//   - LG1: a changed(Property, Context) / added(Property, Context) / removed(Property, BValue, Context) body that does
//     anything besides super calls, and whose first statement after them is not an if whose condition calls
//     isRunning() (or Sys.atSteadyState()). The framework fires these callbacks whenever the slot map is initialized, during decode too, not
//     only while running (corpus B1232 §1232.2); the taught first line is if (!isRunning()) return;.
//   - LG2: a no-argument stationStarted() that does work besides super.stationStarted() in a class with no
//     no-argument started(), serviceStarted() or descendantsStarted(): the station calls it once at boot, never for a component added to a running station.
package lifecycleguard

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-lifecycle-guard"

// Spec registers the rule: [--strict] [--] <src-root>, WARN rows (exit 1 under --strict).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.One,
		Usage:      "usage: lint-lifecycle-guard.sh [--strict] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	where = " (types/logic.md § Tridium rt idioms)"
	lg1   = "LG1 %s() acts with no isRunning() guard: make its first statement if (!isRunning()) return;" + where
	lg2   = "LG2 stationStarted() does startup work and the class has no started(): it runs once at station boot, never for a component added to a running station; move the work to started()" + where
)

// callbacks are the post-change callbacks and their parameter types (BComplex/BComponent signatures).
var callbacks = map[string][]string{
	"changed": {"Property", "Context"},
	"added":   {"Property", "Context"},
	"removed": {"Property", "BValue", "Context"},
}

// startCallbacks run on every component start, a drop into a running station included: started(), a service's
// serviceStarted() (BAbstractService.started calls it) and descendantsStarted().
var startCallbacks = map[string]bool{"started": true, "serviceStarted": true, "descendantsStarted": true}

// Check reports LG1 at each unguarded callback and LG2 at each stationStarted-only class, at the method name.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	decls := javasrc.Decls(toks)
	// started is the set of classes (by enclosing body) that declare a no-argument start callback.
	started := map[string]bool{}
	for _, d := range decls {
		if startCallbacks[d.Name] && len(d.Params) == 0 {
			started[owner(d)] = true
		}
	}
	var out []lint.Finding
	for _, d := range decls {
		body := toks[d.Body[0]:d.Body[1]]
		switch {
		case d.Name == "stationStarted" && len(d.Params) == 0:
			if acts(body) && !started[owner(d)] {
				out = append(out, lint.Finding{Line: d.Line + 1, Msg: lg2})
			}
		case params(d, callbacks[d.Name]):
			if first := firstAction(body); first != nil && !guard(first) {
				out = append(out, lint.Finding{Line: d.Line + 1, Msg: strings.Replace(lg1, "%s", d.Name, 1)})
			}
		}
	}
	return out, nil
}

// owner keys a declaration by its enclosing class; an anonymous body is never another class's started().
func owner(d javasrc.Decl) string {
	if d.Anonymous {
		return "anonymous " + d.Class
	}
	return d.Class
}

// params: d's parameter types end with the wanted simple names (javax.baja.sys.Property matches Property).
func params(d javasrc.Decl, want []string) bool {
	if want == nil || len(d.Params) != len(want) {
		return false
	}
	for i, p := range d.Params {
		if p != want[i] && !strings.HasSuffix(p, "."+want[i]) {
			return false
		}
	}
	return true
}

// statements splits a body into its top-level statements: a block statement (if, for, ...) runs to the end of its
// last block or its terminating semicolon at depth 0, as the next statement starts after it.
func statements(body []javasrc.Token) [][]javasrc.Token {
	var out [][]javasrc.Token
	depth, start := 0, 0
	for k, t := range body {
		switch t.Text {
		case "(", "[", "{":
			depth++
		case ")", "]":
			depth--
		case "}":
			if depth--; depth == 0 && !elseFollows(body, k) {
				out, start = append(out, body[start:k+1]), k+1
			}
		case ";":
			if depth == 0 {
				out, start = append(out, body[start:k+1]), k+1
			}
		}
	}
	if start < len(body) {
		out = append(out, body[start:])
	}
	return out
}

// elseFollows keeps an if-else chain (and a try's catch/finally) one statement.
func elseFollows(body []javasrc.Token, k int) bool {
	if k+1 >= len(body) {
		return false
	}
	switch body[k+1].Text {
	case "else", "catch", "finally":
		return true
	}
	return false
}

// superCall: the statement is super.<name>(...);.
func superCall(s []javasrc.Token) bool {
	return len(s) >= 5 && s[0].Text == "super" && s[1].Text == "." && s[3].Text == "(" && s[len(s)-1].Text == ";"
}

// firstAction is the first statement that is not a super call; nil when the body holds nothing else.
func firstAction(body []javasrc.Token) []javasrc.Token {
	for _, s := range statements(body) {
		if !superCall(s) {
			return s
		}
	}
	return nil
}

// acts: the body holds a statement besides super calls.
func acts(body []javasrc.Token) bool { return firstAction(body) != nil }

// guard: the statement is an if whose parenthesized condition calls isRunning (on this or on any receiver) or
// Sys.atSteadyState() (the station finished loading, so no decode is in flight: a stronger form Tridium uses).
func guard(s []javasrc.Token) bool {
	if len(s) < 2 || s[0].Text != "if" || s[1].Text != "(" {
		return false
	}
	depth := 0
	for k := 1; k < len(s); k++ {
		switch s[k].Text {
		case "(":
			depth++
		case ")":
			if depth--; depth == 0 {
				return false
			}
		case "isRunning", "atSteadyState":
			if k+1 < len(s) && s[k+1].Text == "(" {
				return true
			}
		}
	}
	return false
}
