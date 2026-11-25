// Package wbusability is lint-wb-usability: the narrow, mechanical part of the lint candidate in types/wb-widgets.md
// (UX rule). A -wb view that holds a station reference as a BOrd, in a file that never offers a station-component
// picker, leaves the operator typing an ORD by hand. toolbelt/lint-wb-usability.sh is the wrapper and its header is
// the user-facing contract.
//
// A per-file rule over *.java, read as javasrc tokens (comments blanked, a literal one token). A class whose
// superclass is BWbComponentView (a qualified name counts by its last part) WARNs once, at the class name, when it
// holds a BOrd member and the FILE has neither a BComponentChooser nor a targetType facet. A BOrd member is a
// non-static field of type BOrd or BOrd[], or a class-level @NiagaraProperty whose type is "BOrd" (or "baja:Ord").
// The chooser is an identifier in code, never in a comment or a string, and never in an import line (importing it
// is not opening it); targetType counts as an identifier or as text inside a literal (the facet key). A comment
// `lint-wb-usability: waived <reason>` on the class line or in the five lines above it waives the class; the
// reason is required and a string that holds the same text is not a waiver.
//
// What it does not judge: whether the BOrd is edited by the user or only held, whether the controller that opens
// the chooser lives in another class or file, drag-drop from the nav tree, the status table, and a view that
// extends a subclass of BWbComponentView or a manager base. Those need the semantic reading the earlier triage
// deferred, so this rule is advisory and biased to the false negative.
package wbusability

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-wb-usability"

// Spec registers the rule: [--strict] [--] <wb-src-root>, WARN rows (exit 1 under --strict).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.One,
		Usage:      "usage: lint-wb-usability.sh [--strict] <wb-src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no *.java to scan (K20: wrong path or empty scaffold)"},
		// A -ux or -rt root has no view to judge; a run over a whole module tree must not fail on its parts.
		OptionalInAll: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

const (
	base  = "BWbComponentView"
	taken = "but its file has no BComponentChooser and no targetType facet: the user can only type the ORD; " +
		"let the Add and Edit flow pick it from the station (types/wb-widgets.md UX rule)"
	// waiverWindow is how many lines above the class line a waiver comment may stand (room for annotations).
	waiverWindow = 5
)

// waiver: the marker and a reason of at least one word.
var waiver = regexp.MustCompile(`lint-wb-usability:\s*waived\s+\S`)

// Check reports each class that extends BWbComponentView with a BOrd member and no picker in the file.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	if picker(toks) {
		return nil, nil
	}
	var out []lint.Finding
	for _, c := range javasrc.Classes(toks) {
		if c.Extends != base {
			continue
		}
		member := ordMember(toks, c)
		if member == "" || waived(f, c.Line) {
			continue
		}
		out = append(out, lint.Finding{Line: c.Line + 1, Msg: c.Name + " extends " + base + " and holds a BOrd (" + member + "), " + taken})
	}
	return out, nil
}

// picker: the file names a BComponentChooser in code outside an import, or a targetType facet.
func picker(toks []javasrc.Token) bool {
	for k := 0; k < len(toks); k++ {
		t := toks[k]
		switch {
		case t.Kind == javasrc.TokIdent && t.Text == "import":
			for k < len(toks) && toks[k].Text != ";" {
				k++
			}
		case t.Kind == javasrc.TokIdent && (t.Text == "BComponentChooser" || t.Text == "targetType"):
			return true
		case t.Kind == javasrc.TokLiteral && strings.Contains(t.Text, "targetType"):
			return true
		}
	}
	return false
}

// ordMember describes the first BOrd member of c, or "" when it has none.
func ordMember(toks []javasrc.Token, c javasrc.Class) string {
	if name := ordProperty(toks, c); name != "" {
		return name
	}
	for _, fd := range javasrc.Fields(toks, c.Body) {
		if !fd.Static && (fd.Type == "BOrd" || fd.Type == "BOrd[]") {
			return "field " + fd.Name
		}
	}
	return ""
}

// ordProperty finds a @NiagaraProperty with type "BOrd" among the annotations before c's kind keyword and returns
// "@NiagaraProperty slot <name>" (the slot name when the annotation has one). The walk back stops at the end of
// the previous member: a ";", "{" or "}" outside every parenthesis.
func ordProperty(toks []javasrc.Token, c javasrc.Class) string {
	start := c.At
	for depth := 0; start > 0; start-- {
		switch toks[start-1].Text {
		case ")":
			depth++
		case "(":
			depth--
		case ";", "{", "}":
			if depth == 0 {
				return propertyIn(toks, start, c.At)
			}
		}
	}
	return propertyIn(toks, 0, c.At)
}

// propertyIn scans toks[from:to] for `type = "BOrd"`.
func propertyIn(toks []javasrc.Token, from, to int) string {
	for k := from; k+2 < to; k++ {
		if toks[k].Text != "type" || toks[k+1].Text != "=" || toks[k+2].Kind != javasrc.TokLiteral {
			continue
		}
		if v := strings.Trim(toks[k+2].Text, `"`); v != "BOrd" && v != "baja:Ord" {
			continue
		}
		return "@NiagaraProperty slot" + slotName(toks, from, to, k)
	}
	return ""
}

// slotName is " <name>" of the annotation whose parentheses hold toks[k], or "".
func slotName(toks []javasrc.Token, from, to, k int) string {
	open, depth := -1, 0
	for j := k; j >= from; j-- {
		if toks[j].Text == ")" {
			depth++
		} else if toks[j].Text == "(" {
			if depth == 0 {
				open = j
				break
			}
			depth--
		}
	}
	if open < 0 {
		return ""
	}
	for j, d := open+1, 0; j+2 < to; j++ {
		switch toks[j].Text {
		case "(":
			d++
		case ")":
			if d == 0 {
				return ""
			}
			d--
		}
		if d == 0 && toks[j].Text == "name" && toks[j+1].Text == "=" && toks[j+2].Kind == javasrc.TokLiteral {
			return " " + strings.Trim(toks[j+2].Text, `"`)
		}
	}
	return ""
}

// waived: a comment with the waiver and a reason stands on class line (0-based) or in the lines above it. The
// marker counts only where Strip blanked it, so a string literal holding the same text is not a waiver.
func waived(f *javasrc.File, line int) bool {
	stripped := f.Stripped()
	for i := line; i >= 0 && i >= line-waiverWindow; i-- {
		if waiver.MatchString(f.Raw[i]) && !waiver.MatchString(stripped[i]) {
			return true
		}
	}
	return false
}
