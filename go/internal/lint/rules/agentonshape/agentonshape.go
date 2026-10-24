// Package agentonshape is lint-agent-on-shape: every module-include.xml <on type="..."/> must be exactly
// `module:Type`; a target without the module prefix makes the agent silently fail to appear in Workbench. Port of
// toolbelt/lint-agent-on-shape.sh (awk_lint_each over every module-include.xml); the wrapper keeps the user-facing
// contract and its header documents the rule and its mutation pins.
package agentonshape

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// ID is the lint name printed in every row.
const ID = "lint-agent-on-shape"

// Spec registers the rule with the argument contract of toolbelt/lint-agent-on-shape.sh: the first argument is the
// root (no flag is parsed), left out of n4lint --all, the awk_lint_list walk (a listing error still scans the files listed, then exit 3), the
// K20 row, and awk_lint_each's wording of an unreadable file.
func Spec() lint.Spec {
	return lint.Spec{
		Rule: rule{},
		Mode: lint.Raw,
		// Reason: module-include.xml sits at an artifact's root beside src/, never under a source root, so
		// n4lint --all (source roots) leaves this rule out; report-module runs it on each artifact directory.
		ModuleRoot: true,
		Usage:      "usage: lint-agent-on-shape.sh <src-root>",
		Select:     lint.Selection{Names: []string{"module-include.xml"}},
		// Reason: bash used awk_lint_list + awk_lint_each, which scan the files that were listed (env line, rows, exit 3).
		ContinueOnWalkError: true,
		ShortScanError:      true,
		EmptyError:          lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Fail }

// shape is a well-formed agent-on target: exactly one colon, non-empty module and type parts.
var shape = regexp.MustCompile(`^[A-Za-z0-9_.-]+:[A-Za-z0-9_$]+$`)

// Check reads the file through the shared XML scanner (R8-D12): every <on> element's type attribute, decoded, at the
// element's own line. A malformed module-include.xml is an error naming path:line:col, printed once by the runner (exit
// 3, the other files still scanned), never a guess. Truth fix 2026-10-24 over the awk, which read text line by line: it took only the last
// <on per line, never a single-quoted value, counted an <on> inside a comment or CDATA, reported entity text undecoded
// and an empty type="" as clean, and read a malformed file as if it were well-formed.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	doc, err := xmlsrc.Open(f.Path)
	if err != nil {
		return nil, err
	}
	var out []lint.Finding
	elems := doc.Elements()
	for _, i := range doc.Find("on") {
		if val, ok := elems[i].Attr("type"); ok && !shape.MatchString(val) {
			out = append(out, lint.Finding{Line: elems[i].Line, Msg: "malformed agent-on target (need module:Type): " + val})
		}
	}
	return out, nil
}
