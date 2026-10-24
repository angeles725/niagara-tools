// Package recoverypath is the lint-recovery-path rule: a protection or heat output slot (a name ending in
// Out, Resistance, Heat or Resistencia) written OFF (`getX().setValue(false)` or `setX(false)`) only in
// methods other than started, atSteadyState, changed, enable or start FAILs: nothing resets it when the
// mode that gated it clears (the PANCCADIA stranded heater). Port of the awk in
// toolbelt/lint-recovery-path.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins). Per-file rule over javasrc.Methods; a file's rows come in gawk's
// `for (slot in a)` order.
package recoverypath

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/awkorder"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
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
	// a safe-off write at an identifier boundary: getX().setValue(false) (group 2) or setX(false) (group 3)
	offWrite = rx(`(^|[^A-Za-z0-9_$])(?:get([A-Za-z][A-Za-z0-9_]*)\(\)\.setValue|set([A-Za-z][A-Za-z0-9_]*))\([[:space:]]*false[[:space:]]*\)`)
	outSlot  = rx(`(Out|Resistance|Heat|Resistencia)$`)
	resetter = rx(`^(` + strings.Join(resets, "|") + `)$`)
)

// resets are the methods whose safe-off write counts as the unconditional reset, in the order the row names them.
var resets = []string{"started", "atSteadyState", "changed", "enable", "start"}

// guard is the first write of a slot outside a reset method.
type guard struct {
	line   int
	method string
}

// Check FAILs every slot of the file written OFF outside a reset method and never inside one.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := linescan.Code(f) // comments blanked and literal contents removed: writes are code only
	method := make([]string, len(code))
	for _, m := range f.Methods() { // the shared method scanner (javasrc.Methods): name and body lines
		for i := m.Start - 1; i < m.End; i++ {
			method[i] = m.Name
		}
	}
	reset := map[string]bool{}
	guards := map[string]guard{}
	var order []string // guarded slots in first-write order, the order gawk inserted them
	for i, s := range code {
		m := method[i]
		if m == "" {
			m = "unknown"
		}
		for _, slot := range offSlots(s) {
			if resetter.MatchString(m) {
				reset[slot] = true
			} else if _, seen := guards[slot]; !seen {
				guards[slot] = guard{i + 1, m}
				order = append(order, slot)
			}
		}
	}
	var out []lint.Finding
	for _, slot := range awkorder.Order(order) {
		if g := guards[slot]; !reset[slot] {
			out = append(out, lint.Finding{Line: g.line, Msg: "LRP: " + slot + " has guarded-only safe-off in method \"" +
				g.method + "\" -- no unconditional reset in " + strings.Join(resets, "/")})
		}
	}
	return out, nil
}

// offSlots are the protection slots line s writes OFF, in line order: every getX().setValue(false) and
// setX(false) starting at an identifier boundary whose X ends in Out, Resistance, Heat or Resistencia.
func offSlots(s string) []string {
	var out []string
	for _, m := range offWrite.FindAllStringSubmatch(s, -1) {
		if name := m[2] + m[3]; outSlot.MatchString(name) {
			out = append(out, name)
		}
	}
	return out
}
