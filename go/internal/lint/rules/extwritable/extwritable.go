// Package extwritable is lint-ext-writable-shape: an OPERATOR complex property (BStatusNumeric,
// BStatusBoolean, BStatusEnum) that no @NiagaraAction writes is a hazardous external-write target. Port of
// the awk in toolbelt/lint-ext-writable-shape.sh; the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins.
//
// It is a per-file lint.Rule: a property is exempt only through the actions and do<Action> bodies of its
// own file. Matching keeps gawk's C-locale byte semantics: toupper and tolower fold ASCII only.
package extwritable

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts; rows print the check name instead.
const ID = "lint-ext-writable-shape"

// check is the check column of every row, as the bash lint printed it.
const check = "ext-writable-shape"

// Spec registers the rule with the argument contract of toolbelt/lint-ext-writable-shape.sh: flags, then
// one root (later arguments ignored), a usage error prints the usage line alone and exits 3, and a root
// with no .java file is the K20 ERROR row. The walk stops on an unreadable directory (scan_files_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-ext-writable-shape.sh [--strict] <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Check: check, Msg: "no Java sources found (K20: wrong path or empty scaffold)"},
		BareUsage:  true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	typeAttr  = regexp.MustCompile(`type[[:space:]]*=[[:space:]]*"([^"]*)"`)
	flagsAttr = regexp.MustCompile(`flags[[:space:]]*=[[:space:]]*`)
	flagsVal  = regexp.MustCompile(`^[^,)]*`)
	// complexType is the awk's recognizer; its optional prefixes make it a substring match of the three stems.
	complexType = regexp.MustCompile(`([Bb]aja:)?(B)?[Ss]tatus[Nn]umeric|([Bb]aja:)?(B)?[Ss]tatus[Bb]oolean|([Bb]aja:)?(B)?[Ss]tatus[Ee]num`)
)

// Check harvests the file's @NiagaraAction names, marks every slot a do<Action> body writes, then reports
// each OPERATOR complex @NiagaraProperty that neither a body nor an action name exempts.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := f.Stripped()
	code, _ := javasrc.BlankLiterals(lines, nil) // a "(", marker or setX( inside a literal is not code
	actions, handlers := harvestActions(lines, code)
	exempt := map[string]bool{}
	if len(handlers) > 0 {
		for _, m := range f.Methods() {
			if !handlers[m.Name] {
				continue
			}
			var body strings.Builder
			for i := m.Start; i <= m.End; i++ {
				body.WriteByte(' ')
				body.WriteString(code[i-1])
			}
			scanWrites(body.String(), exempt)
		}
	}
	var out []lint.Finding
	annotations(lines, code, "@NiagaraProperty", func(buf string, line int) {
		if msg, ok := checkProp(buf, actions, exempt); ok {
			out = append(out, lint.Finding{Check: check, Line: line, Msg: msg})
		}
	})
	return out, nil
}

// checkProp returns the row message of one joined @NiagaraProperty, or false when it has no name or type,
// is not complex, is not OPERATOR, or is exempt.
func checkProp(buf string, actions, exempt map[string]bool) (string, bool) {
	name := attr(nameAttr, buf)
	if name == "" {
		return "", false
	}
	typ := attr(typeAttr, buf)
	if typ == "" || !complexType.MatchString(typ) {
		return "", false
	}
	flags := ""
	if loc := flagsAttr.FindStringIndex(buf); loc != nil {
		flags = flagsVal.FindString(buf[loc[1]:])
	}
	if !strings.Contains(flags, "OPERATOR") && !strings.Contains(flags, `"o"`) {
		return "", false
	}
	if exempt[name] {
		return "", false
	}
	uc := upperFirst(name)
	if actions["set"+uc] || actions["apply"+uc] || actions[name+"Cmd"] {
		return "", false
	}
	return name + ": OPERATOR " + typ + " with no writing action -- external oBIX write must use the child leaf .../" +
		name + "/value (bare <real>, B826) or add an OPERATOR action (B822)", true
}

// attr is the quoted value of the first match of re in buf, or "".
func attr(re *regexp.Regexp, buf string) string {
	if m := re.FindStringSubmatch(buf); m != nil {
		return m[1]
	}
	return ""
}
