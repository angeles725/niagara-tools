// Package wbfilechooser is the lint-wb-file-chooser rule: a -wb BWbFieldEditor.dialog(...BOrd.NULL) that
// opens the file-system chooser, in a file with no station-space picker. Port of the check in
// toolbelt/lint-wb-file-chooser.sh (the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins).
package wbfilechooser

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-wb-file-chooser"

// Spec registers the rule with the argument contract of toolbelt/lint-wb-file-chooser.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.First,
		Usage:  "usage: lint-wb-file-chooser.sh [--strict] <wb-src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_strip_run (awk_lint_list), which scans the partial list (env line, rows, exit 3).
		ContinueOnWalkError: true,
	}
}

const (
	dialog = "BWbFieldEditor.dialog("
	null   = "BOrd.NULL"
)

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// Check, over the code before each line's first "//": a file that calls the dialog and names BOrd.NULL,
// with neither BComponentChooser nor targetType, reports the first dialog line that also holds
// BOrd.NULL, else its first dialog line.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	first, both, hasNull := 0, 0, false
	for i, ln := range f.Raw {
		code := linescan.Cut(ln)
		if strings.Contains(code, "BComponentChooser") || strings.Contains(code, "targetType") {
			return nil, nil
		}
		isNull := strings.Contains(code, null)
		hasNull = hasNull || isNull
		if strings.Contains(code, dialog) {
			if first == 0 {
				first = i + 1
			}
			if both == 0 && isNull {
				both = i + 1
			}
		}
	}
	if first == 0 || !hasNull {
		return nil, nil
	}
	at := both
	if at == 0 {
		at = first
	}
	return []lint.Finding{{Line: at, Msg: linescan.Trimmed(f.Raw[at-1])}}, nil
}
