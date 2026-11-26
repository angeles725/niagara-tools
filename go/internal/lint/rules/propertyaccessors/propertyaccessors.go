// Package propertyaccessors is lint-property-accessors: a B* class that declares a slot by hand
// (`public static final Property x = newProperty(...)`) must also declare a public get<X>/is<X> and a public set<X>.
// When the station loads the type, the introspector reads every Property field and needs both accessors on the
// class; with either missing it logs `GRAVE [sys.schema] ... No setter for property (x)` / `Cannot load type for
// ...` and every later use throws NoClassDefFoundError. The module compiles, signs and passes every build gate
// (measured on Niagara 4.14.0.162: a BPointExtension with three newProperty slots and getters only was unloadable;
// public setters fixed it). Slot-o-Matic generated code always has both, so only hand-written slots are caught.
// toolbelt/lint-property-accessors.sh is the wrapper and its header is the user-facing contract.
//
// A per-file rule over *.java read as javasrc tokens (comments blanked, a literal one token). Judged: each top-level
// class named B followed by an upper-case letter. Its slots are the static `Property` fields declared directly in
// its body whose initializer starts `newProperty(` (javasrc.Fields: a nested class's fields are not the outer's).
// Its accessors are the methods declared directly in its body (a nested class's methods do not count). One FAIL row
// per missing or non-public accessor, at the line of the Property declaration.
package propertyaccessors

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-property-accessors"

// Spec registers the rule: one <src-root>, FAIL rows (exit 1).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Raw,
		Usage:      "usage: lint-property-accessors.sh <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// consequence is what a missing accessor does at runtime, for the property name p and the class c.
func consequence(p, c string) string {
	return "the station cannot load the type (sys.schema \"No setter for property (" + p + ")\" / \"Cannot load type for " +
		c + "\", NoClassDefFoundError on every use)"
}

// niagara: B followed by an upper-case letter.
func niagara(name string) bool {
	return len(name) > 1 && name[0] == 'B' && name[1] >= 'A' && name[1] <= 'Z'
}

// method is one method name declared directly in a class body: seen, and whether any overload is public.
type method struct{ public bool }

// methods are the methods declared directly in the class body [body[0], body[1]) by name. A member starts after
// the previous ";" or "}" at depth 0 (braces and parentheses counted together, so an annotation's arguments and a
// method body are skipped whole); its name is the identifier before its first "(" unless the header holds an "="
// (a field initializer) or the name follows "@" or "new".
func methods(toks []javasrc.Token, body [2]int) map[string]method {
	out := map[string]method{}
	start, depth := body[0], 0
	for j := body[0]; j < body[1]; j++ {
		switch toks[j].Text {
		case "(", "{":
			if toks[j].Text == "(" && depth == 0 && j > start && isMethodName(toks, start, j) {
				m := out[toks[j-1].Text]
				for _, t := range toks[start : j-1] {
					m.public = m.public || t.Text == "public"
				}
				out[toks[j-1].Text] = m
			}
			depth++
		case ")", "}":
			if depth--; depth == 0 && toks[j].Text == "}" {
				start = j + 1
			}
		case ";":
			if depth == 0 {
				start = j + 1
			}
		}
	}
	return out
}

// isMethodName: toks[j-1] names a method whose header starts at start.
func isMethodName(toks []javasrc.Token, start, j int) bool {
	if toks[j-1].Kind != javasrc.TokIdent || j-1 == start && toks[j-1].Text == "new" {
		return false
	}
	if j-2 >= start && (toks[j-2].Text == "@" || toks[j-2].Text == "new" || toks[j-2].Text == ".") {
		return false
	}
	for _, t := range toks[start : j-1] {
		if t.Text == "=" {
			return false
		}
	}
	return true
}

// slot reports whether f is `static Property <name> = newProperty(`.
func slot(toks []javasrc.Token, f javasrc.Field) bool {
	return f.Static && (f.Type == "Property" || strings.HasSuffix(f.Type, ".Property")) &&
		f.Init[0]+1 < len(toks) && toks[f.Init[0]].Text == "newProperty" && toks[f.Init[0]+1].Text == "("
}

// topLevel reports whether classes[i] lies inside no other class body.
func topLevel(classes []javasrc.Class, i int) bool {
	for k, d := range classes {
		if k != i && d.Body[0] <= classes[i].At && classes[i].At < d.Body[1] {
			return false
		}
	}
	return true
}

// Check reports each missing or non-public accessor at its Property declaration, in declaration order.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	classes := javasrc.Classes(toks)
	var out []lint.Finding
	for i, c := range classes {
		if c.Kind != "class" || !niagara(c.Name) || !topLevel(classes, i) {
			continue
		}
		var have map[string]method
		for _, fld := range javasrc.Fields(toks, c.Body) {
			if !slot(toks, fld) {
				continue
			}
			if have == nil {
				have = methods(toks, c.Body)
			}
			out = append(out, judge(c.Name, fld, have)...)
		}
	}
	return out, nil
}

// judge is the rows for one slot: its getter (get<X> or is<X>), then its setter (set<X>).
func judge(class string, fld javasrc.Field, have map[string]method) []lint.Finding {
	x := strings.ToUpper(fld.Name[:1]) + fld.Name[1:]
	var out []lint.Finding
	row := func(msg string) {
		out = append(out, lint.Finding{Line: fld.Line + 1, Msg: "property " + fld.Name + " of " + class + ": " + msg +
			": " + consequence(fld.Name, class) + " (types/structure.md § Property accessors)"})
	}
	get, is := have["get"+x], have["is"+x]
	_, hasGet := have["get"+x]
	_, hasIs := have["is"+x]
	switch {
	case get.public || is.public:
	case hasGet || hasIs:
		row("its getter is not public; declare public get" + x + "() or is" + x + "()")
	default:
		row("no public get" + x + "() or is" + x + "()")
	}
	if set, ok := have["set"+x]; !ok {
		row("no public set" + x + "(...)")
	} else if !set.public {
		row("set" + x + "(...) is not public")
	}
	return out
}
