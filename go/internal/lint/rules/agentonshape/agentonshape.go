// Package agentonshape is lint-agent-on-shape: every module-include.xml <on type="..."/> must be exactly
// `module:Type`; a target without the module prefix makes the agent silently fail to appear in Workbench. Port of
// toolbelt/lint-agent-on-shape.sh (awk_lint_each over every module-include.xml); the wrapper keeps the user-facing
// contract and its header documents the rule and its mutation pins.
package agentonshape

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
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

// Check is the awk program, line by line: the value of the line's last `<on<space>` tag that holds a quoted type=,
// and inside it the last `type="` before the tag's ">" (the GNU sed capture it replaced).
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	for i, ln := range f.Raw {
		if val := onType(ln); val != "" && !shape.MatchString(val) {
			out = append(out, lint.Finding{Line: i + 1, Msg: "malformed agent-on target (need module:Type): " + val})
		}
	}
	return out, nil
}

// onType is the awk's backward scan of one line; "" when no <on tag on it holds a quoted type=.
func onType(ln string) string {
	for p := len(ln) - 4; p >= 0; p-- { // awk p = length-3 .. 1, 1-based
		if !strings.HasPrefix(ln[p:], "<on") || p+3 >= len(ln) || !isSpace(ln[p+3]) {
			continue
		}
		tag := ln[p+4:]
		gt := strings.IndexByte(tag, '>')
		if gt < 0 {
			gt = len(tag)
		}
		for q := gt - 1; q >= 0; q-- {
			if !strings.HasPrefix(tag[q:], `type="`) {
				continue
			}
			v := tag[q+6:]
			if e := strings.IndexByte(v, '"'); e >= 0 {
				return v[:e]
			}
		}
	}
	return ""
}

// isSpace is awk's [[:space:]] in the C locale.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}
