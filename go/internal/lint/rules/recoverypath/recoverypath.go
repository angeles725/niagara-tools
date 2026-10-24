// Package recoverypath is the lint-recovery-path rule: a protection or heat output slot (a name ending in
// Out, Resistance, Heat or Resistencia) written OFF (`getX().setValue(false)` or `setX(false)`) only in
// methods other than started, atSteadyState, changed, enable or start FAILs: nothing resets it when the
// mode that gated it clears (the PANCCADIA stranded heater). Port of the awk in
// toolbelt/lint-recovery-path.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pin). Per-file rule; a file's rows come in gawk's `for (slot in a)` order.
package recoverypath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/awkorder"
)

// ID is the lint name printed in every row.
const ID = "lint-recovery-path"

// Spec registers the rule with the argument contract of toolbelt/lint-recovery-path.sh: no flag is parsed
// and only the first argument is read (Raw); an incomplete walk or a root with no *.java stops the lint
// (scan_files_nonempty_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Raw,
		Usage:      "usage: lint-recovery-path.sh <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	callName  = rx(`[a-z][A-Za-z0-9_]+[[:space:]]*\(`)
	leadName  = rx(`^[a-z][A-Za-z0-9_]+`)
	keyword   = rx(`^(if|for|while|switch|try|catch|super|this|return|throw|new|else|assert|synchronized|do)$`)
	getterOff = rx(`get[A-Za-z][A-Za-z0-9_]*\(\)\.setValue\([[:space:]]*false[[:space:]]*\)`)
	setterOff = rx(`set[A-Za-z][A-Za-z0-9_]*\([[:space:]]*false[[:space:]]*\)`)
	outSlot   = rx(`(Out|Resistance|Heat|Resistencia)$`)
	resetter  = rx(`^(started|atSteadyState|changed|enable|start)$`)
)

// guard is the first write of a slot outside a reset method.
type guard struct {
	line   int
	method string
}

// Check FAILs every slot of the file written OFF outside a reset method and never inside one.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := make([]string, len(f.Raw)) // each line cut at its first "//", as the awk's sub(/\/\/.*$/, "")
	for i, ln := range f.Raw {
		code[i], _, _ = strings.Cut(ln, "//")
	}
	depth, method := 0, ""
	reset := map[string]bool{}
	guards := map[string]guard{}
	var order []string // guarded slots in first-write order, the order gawk inserted them
	for i, s := range code {
		open := strings.Count(s, "{")
		if depth == 1 && open > 0 {
			method = enclosing(code, i)
		}
		if depth += open - strings.Count(s, "}"); depth <= 1 {
			method = ""
		}
		slot := offSlot(s)
		if slot == "" {
			continue
		}
		m := method
		if m == "" {
			m = "unknown"
		}
		if resetter.MatchString(m) {
			reset[slot] = true
		} else if _, seen := guards[slot]; !seen {
			guards[slot] = guard{i + 1, m}
			order = append(order, slot)
		}
	}
	var out []lint.Finding
	for _, slot := range awkorder.Order(order) {
		if g := guards[slot]; !reset[slot] {
			out = append(out, lint.Finding{Line: g.line, Msg: "LRP: " + slot + " has guarded-only safe-off in method \"" +
				g.method + "\" -- no unconditional reset in started/atSteadyState/enable"})
		}
	}
	return out, nil
}

// enclosing names the method a "{" on line i opens: the first lower-case call name on that line or one of
// the 5 above it that is not a keyword ("" when there is none).
func enclosing(code []string, i int) string {
	for j := i; j >= i-5 && j >= 0; j-- {
		if loc := callName.FindStringIndex(code[j]); loc != nil {
			if nm := leadName.FindString(code[j][loc[0]:]); !keyword.MatchString(nm) {
				return nm
			}
		}
	}
	return ""
}

// offSlot is the protection slot line s writes OFF, or "": the first getX().setValue(false), else the first
// setX(false), whose X ends in Out, Resistance, Heat or Resistencia.
func offSlot(s string) string {
	if loc := getterOff.FindStringIndex(s); loc != nil {
		name := s[loc[0]+len("get"):]
		name = name[:strings.Index(name, "().setValue")]
		if outSlot.MatchString(name) {
			return name
		}
	}
	if loc := setterOff.FindStringIndex(s); loc != nil {
		name := s[loc[0]+len("set"):]
		name = name[:strings.IndexByte(name, '(')]
		if outSlot.MatchString(name) {
			return name
		}
	}
	return ""
}
