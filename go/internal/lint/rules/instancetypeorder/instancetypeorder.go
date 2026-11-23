// Package instancetypeorder is the lint-instance-type-order rule (issue #847): a Niagara type class (one
// annotated @NiagaraType, or one holding a Slot-o-Matic `static ... Type TYPE = Sys.loadType(...)` field) that
// declares a static INSTANCE field AFTER its static TYPE field. A 4.14 lab station then refuses the type with
// "INSTANCE field is null; insure getType() comes last" (issue #847; the station's own source was not read).
// Pure tests, build, verify and report all pass; only a station load shows it, so this is the gate. A class
// with no INSTANCE, or INSTANCE before TYPE, is clean.
//
// Source is read as the sibling rules read it (javasrc.Tokens over the comment-stripped lines): a comment, a
// string or text block, a local variable, a nested type's member and an anonymous class are no field of the
// class. Each class is judged on its own fields (javasrc.Fields), so a nested type never leaks into its
// outer class. FAIL, on the INSTANCE line.
package instancetypeorder

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-instance-type-order"

// Spec registers the rule with the argument contract of toolbelt/lint-instance-type-order.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Each,
		Usage:  "usage: lint-instance-type-order.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

const msg = "static INSTANCE is declared after TYPE: the station refuses to load the type with" +
	" \"INSTANCE field is null; insure getType() comes last\" -- declare INSTANCE before TYPE"

// Check reports, per Niagara type class, each static INSTANCE field declared after its static Type TYPE field,
// on the INSTANCE line.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	var out []lint.Finding
	for _, c := range javasrc.Classes(toks) {
		annotated := javasrc.Annotated(toks, c, "NiagaraType")
		seen := false // a TYPE field of this class stands earlier in the body
		for _, fd := range javasrc.Fields(toks, c.Body) {
			switch {
			case !fd.Static:
			case fd.Name == "TYPE" && isType(fd.Type) && (annotated || loadsType(toks, fd)):
				seen = true
			case fd.Name == "INSTANCE" && seen:
				out = append(out, lint.Finding{Line: fd.Line + 1, Msg: msg})
			}
		}
	}
	return out, nil
}

// isType: a field declared as Type or a qualified …Type (javax.baja.sys.Type).
func isType(typ string) bool { return typ == "Type" || strings.HasSuffix(typ, ".Type") }

// loadsType: the field's initializer holds Sys.loadType, the Slot-o-Matic shape, which marks a type class that carries no @NiagaraType.
func loadsType(toks []javasrc.Token, fd javasrc.Field) bool {
	for i := fd.Init[0]; i+2 < fd.Init[1]; i++ {
		if toks[i].Text == "Sys" && toks[i+1].Text == "." && toks[i+2].Text == "loadType" {
			return true
		}
	}
	return false
}
