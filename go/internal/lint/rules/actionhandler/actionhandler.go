// Package actionhandler is lint-action-handler: every @NiagaraAction(name = "x") of a class needs its do<X>
// handler, or the station refuses the type at load. The introspector looks up do<X>() / do<X>(P), then
// do<X>(Context) / do<X>(P, Context) among the class's PUBLIC methods and throws "Missing action do method"
// when neither exists, "Do action method must be not be static" for a static one and "Action method and do
// method have mismatched signature" when its return type differs from the action wrapper's
// (com/tridium/sys/schema/ComplexIntrospector.java:316-343; corpus B1231 §1231.4, B1417 §1417.3 j). Slot-o-Matic
// and the jar task succeed, so only a station load shows it: this is the gate. toolbelt/lint-action-handler.sh
// is the wrapper and its header is the user-facing contract.
//
// A per-file rule over *.java read as javasrc tokens (comments blanked, a literal one token). Rows: FAIL a
// missing or non-public do method, a static one, a return type that differs from the action's `returnType` (void
// when none); WARN doX(..., Context) beside a no-Context doX(...) of the same parameters, which never runs (the
// no-Context one is found first; ledger G1); WARN `this::doX` or `new Thread(() -> doX(` of a declared action,
// which skips the action pipeline (course F3; a `post(Action...)` override is lint-async-blocking's). A handler
// inherited from a class in ANOTHER file is not seen (a superclass in the same file is).
package actionhandler

import (
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-action-handler"

// Spec registers the rule with the argument contract of toolbelt/lint-action-handler.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Each,
		Usage:  "usage: lint-action-handler.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Check reports, per class, the handler rows, the pair rows and the method-reference rows in line order.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	classes := javasrc.Classes(toks)
	decls := javasrc.Decls(toks)
	var out []lint.Finding
	for _, c := range classes {
		ms := doMethods(toks, c)
		out = append(out, handlerRows(c, Actions(toks, c), withAncestors(toks, classes, c, ms))...)
		out = append(out, pairRows(ms)...)
		out = append(out, refRows(toks, classes, c, decls)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Line < out[j].Line })
	return out, nil
}
