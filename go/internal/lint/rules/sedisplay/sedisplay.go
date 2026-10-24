// Package sedisplay is the lint-se-display rule: a -se part (loaded on the headless station) that imports
// or instantiates a display-dependent AWT/Swing class. Port of the awk in toolbelt/lint-se-display.sh (the
// wrapper keeps the user-facing contract and its header documents the rule and its mutation pins).
package sedisplay

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-se-display"

// Spec registers the rule with the argument contract of toolbelt/lint-se-display.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-se-display.sh <se-src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

// display is every text that names a display class; each is a plain substring of the code.
var display = []string{
	"import javax.swing.JFrame", "import javax.swing.JDialog", "import javax.swing.JWindow",
	"import javax.swing.JApplet", "import javax.swing.JOptionPane",
	"import java.awt.Frame", "import java.awt.Window", "import java.awt.Dialog",
	"new JFrame", "new JDialog", "new JWindow",
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// Check reports every line whose code (linescan.Code: no comment, Javadoc or literal text) holds one of
// the display texts.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i, code := range linescan.Code(f) {
		for _, d := range display {
			if strings.Contains(code, d) {
				out = append(out, lint.Finding{Line: i + 1, Msg: "display class: " + linescan.Source(f.Raw[i])})
				break
			}
		}
	}
	return out, nil
}
