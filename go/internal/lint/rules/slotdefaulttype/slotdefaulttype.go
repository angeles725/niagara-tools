// Package slotdefaulttype is lint-slot-default-type: an @NiagaraProperty of a primitive or String type whose
// defaultValue literal has another Java type. Slot-o-Matic copies defaultValue verbatim into
// newProperty(flags, <default>, facets); Java overload resolution then picks newProperty(int, boolean|int|long|float|
// double|String|BValue, ...) by the literal's own type (BComplex.java:690-750), so the property holds a value of that
// type. When the type is introspected, ComplexIntrospector widens a BInteger default to a long, float or double slot
// and refuses every other mismatch ("Property default value is of wrong type", ComplexIntrospector.java:199-237): the
// module compiles and the type fails at runtime — the course's "int property with default false compiles and fails"
// (corpus B1417 §1417.3 row l).
// toolbelt/lint-slot-default-type.sh is the wrapper and its header is the user-facing contract.
//
// Only literals the rule can type are judged: true/false, an integer literal (long with an L suffix), a floating
// literal (float with an f suffix), a string literal, and B<Boolean|Integer|Long|Float|Double|String>.<make(...) or a
// constant>. Any other expression (a constant name, new BStatusNumeric()) is not judged.
package slotdefaulttype

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-slot-default-type"

// Spec registers the rule: one <src-root>, FAIL rows (exit 1).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Raw,
		Usage:      "usage: lint-slot-default-type.sh <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

var (
	nameAttr    = regexp.MustCompile(`\bname\s*=\s*"([^"]*)"`)
	typeAttr    = regexp.MustCompile(`\btype\s*=\s*"([^"]*)"`)
	defaultAttr = regexp.MustCompile(`\bdefaultValue\s*=\s*"((?:[^"\\]|\\.)*)"`)

	intLit   = regexp.MustCompile(`^-?\s*(0[xX][0-9a-fA-F_]+|0[bB][01_]+|[0-9][0-9_]*)$`)
	longLit  = regexp.MustCompile(`^-?\s*(0[xX][0-9a-fA-F_]+|0[bB][01_]+|[0-9][0-9_]*)[lL]$`)
	floatLit = regexp.MustCompile(`^-?\s*([0-9][0-9_]*\.?[0-9_]*|\.[0-9][0-9_]*)([eE][+-]?[0-9]+)?[fF]$`)
	dblLit   = regexp.MustCompile(`^-?\s*(([0-9][0-9_]*\.[0-9_]*|\.[0-9][0-9_]*)([eE][+-]?[0-9]+)?[dD]?|[0-9][0-9_]*([eE][+-]?[0-9]+[dD]?|[dD]))$`)
	strLit   = regexp.MustCompile(`^"(?:[^"\\]|\\.)*"$`)
	bValue   = regexp.MustCompile(`^B(Boolean|Integer|Long|Float|Double|String)\.(make\(.*\)|[A-Z_][A-Z0-9_]*)$`)
)

// slots is what each declared type accepts: its primitive (or String) literal kind and its B-value kind.
var slots = map[string]struct{ prim, bval string }{
	"int":     {"int", "BInteger"},
	"long":    {"long", "BLong"},
	"float":   {"float", "BFloat"},
	"double":  {"double", "BDouble"},
	"boolean": {"boolean", "BBoolean"},
	"String":  {"String", "BString"},
	"BString": {"String", "BString"},
}

// widened are the slot types whose introspection turns a BInteger default into their own value (an int literal fits).
var widened = map[string]bool{"long": true, "float": true, "double": true}

// stored is the B type newProperty stores for a literal kind.
var stored = map[string]string{"int": "BInteger", "long": "BLong", "float": "BFloat", "double": "BDouble",
	"boolean": "BBoolean", "String": "BString"}

// kind is the Java type of a default expression, "" when the rule cannot type it.
func kind(expr string) string {
	switch e := strings.TrimSpace(expr); {
	case e == "true" || e == "false":
		return "boolean"
	case intLit.MatchString(e):
		return "int"
	case longLit.MatchString(e):
		return "long"
	case floatLit.MatchString(e):
		return "float"
	case dblLit.MatchString(e):
		return "double"
	case strLit.MatchString(e):
		return "String"
	default:
		if m := bValue.FindStringSubmatch(e); m != nil {
			return "B" + m[1]
		}
	}
	return ""
}

func article(k string) string {
	if strings.ContainsRune("aeiouAEIOU", rune(k[0])) {
		return "an " + k
	}
	return "a " + k
}

// Check reads every @NiagaraProperty annotation of the comment-stripped file (literal text kept).
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	javasrc.NiagaraProperties(f.Stripped(), func(text string, line int) {
		ty, dv := typeAttr.FindStringSubmatch(text), defaultAttr.FindStringSubmatch(text)
		if ty == nil || dv == nil {
			return
		}
		slot, ok := slots[ty[1]]
		expr := javasrc.Unescape(dv[1])
		k := kind(expr)
		if !ok || k == "" || k == slot.prim || k == slot.bval || (k == "int" || k == "BInteger") && widened[ty[1]] {
			return
		}
		name := ""
		if m := nameAttr.FindStringSubmatch(text); m != nil {
			name = m[1]
		}
		st := stored[k]
		if st == "" {
			st = k
		}
		out = append(out, lint.Finding{Line: line, Msg: "@NiagaraProperty " + name + ` type="` + ty[1] + `" defaultValue="` + dv[1] +
			`": the default is ` + article(k) + ", so Slot-o-Matic's newProperty(flags, " + expr + ", ...) stores " + article(st) +
			" that is not a " + slot.bval + ": the type fails to load (\"Property default value is of wrong type\") (types/structure.md § Slot defaults)"})
	})
	return out, nil
}
