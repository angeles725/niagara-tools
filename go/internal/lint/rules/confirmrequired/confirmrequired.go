// Package confirmrequired is the lint-confirm-required rule (types/actions.md § 3): a @NiagaraAction whose
// name says it destroys or rewrites data (a whole camelCase word of it is reset, clear, delete, remove, purge,
// wipe, factory, restart, reboot or overwrite) must declare Flags.CONFIRM_REQUIRED, so Workbench and the
// bajaux invokeAction ask before running it. The flag is a client prompt, not authorization (Flags.java:190,
// InvokeActionCommand.java:42-51), so a missing flag is a FAIL and the rule says nothing about who may invoke.
//
// Source is read as the sibling rules read it: javasrc.Tokens over the comment-stripped lines, so a comment,
// a Javadoc line, a string or a text block holding "@NiagaraAction(...)" is no annotation. An annotation is
// its name (also qualified), a "(" (on the same or a later line) and the arguments up to the matching ")".
// A `// lint-confirm-required: waived <reason>` comment on the annotation's line, or on a // comment line right
// above it, waives it; a marker with no reason is itself a FAIL. FAIL, on the "@" line.
package confirmrequired

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-confirm-required"

// Spec registers the rule with the argument contract of toolbelt/lint-confirm-required.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Each,
		Usage:  "usage: lint-confirm-required.sh [--strict] <src-root>...",
		Select: lint.Selection{Names: []string{"*.java"}},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// confirmRequired is Flags.CONFIRM_REQUIRED (javax/baja/sys/Flags.java:190).
const confirmRequired = 0x80

// destructive is the heuristic: the words of an action name that mark it destructive.
var destructive = map[string]bool{
	"reset": true, "clear": true, "delete": true, "remove": true, "purge": true,
	"wipe": true, "factory": true, "restart": true, "reboot": true, "overwrite": true,
}

var (
	withReason  = regexp.MustCompile(`lint-confirm-required:[[:space:]]*waived[[:space:]]+[^[:space:]]`)
	marker      = regexp.MustCompile(`lint-confirm-required:[[:space:]]*waived`)
	commentLine = regexp.MustCompile(`^[[:space:]]*//`)
)

const (
	missingMsg = "destructive action without Flags.CONFIRM_REQUIRED -- declare flags = Flags.CONFIRM_REQUIRED" +
		" (with | Flags.OPERATOR when operators may run it) or waive with // lint-confirm-required: waived <reason>"
	bareMsg = "waiver without a reason -- write // lint-confirm-required: waived <reason>"
)

// Check reports each destructive @NiagaraAction that lacks CONFIRM_REQUIRED in its flags, on the line of its
// "@", unless a waiver with a reason covers it.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	toks := javasrc.Tokens(f.Stripped())
	var out []lint.Finding
	for i, t := range toks {
		if t.Kind != javasrc.TokIdent || t.Text != "NiagaraAction" {
			continue
		}
		at := annotationStart(toks, i)
		if at < 0 || i+1 >= len(toks) || toks[i+1].Text != "(" {
			continue
		}
		name, flags := attributes(toks, i+2)
		if !isDestructive(name) || hasConfirm(flags) {
			continue
		}
		line := toks[at].Line
		switch waiver(f.Raw, line) {
		case "reason":
		case "bare":
			out = append(out, lint.Finding{Line: line + 1, Msg: name + ": " + bareMsg})
		default:
			out = append(out, lint.Finding{Line: line + 1, Msg: name + ": " + missingMsg})
		}
	}
	return out, nil
}

// annotationStart is the index of the "@" that starts the annotation whose last identifier is toks[i]
// (a qualified javax.x.Name walks back over its dots and package names), or -1 when no "@" starts it
// (`@interface NiagaraAction`, a type reference).
func annotationStart(toks []javasrc.Token, i int) int {
	for j := i - 1; j >= 0; j -= 2 {
		switch {
		case toks[j].Text == "@":
			return j
		case toks[j].Text != "." || j == 0 || toks[j-1].Kind != javasrc.TokIdent:
			return -1
		}
	}
	return -1
}

// attributes reads the argument list that starts at toks[from] (just after the "(") up to its matching ")":
// the string literal of its name attribute ("" when the value is not one) and the tokens of its flags value.
func attributes(toks []javasrc.Token, from int) (name string, flags []javasrc.Token) {
	depth := 1
	var key string
	var val []javasrc.Token
	finish := func() {
		switch key {
		case "name":
			if len(val) == 1 && val[0].Kind == javasrc.TokLiteral && strings.HasPrefix(val[0].Text, `"`) {
				name = strings.Trim(val[0].Text, `"`)
			}
		case "flags":
			flags = val
		}
		key, val = "", nil
	}
	for k := from; k < len(toks) && depth > 0; k++ {
		t := toks[k]
		switch t.Text {
		case "(", "{", "[":
			depth++
		case ")", "}", "]":
			depth--
		}
		switch {
		case depth == 0:
			finish()
		case depth == 1 && t.Text == ",":
			finish()
		case depth == 1 && key == "" && t.Kind == javasrc.TokIdent && k+1 < len(toks) && toks[k+1].Text == "=":
			key = t.Text
			k++
		case key != "":
			val = append(val, t)
		}
	}
	return name, flags
}
