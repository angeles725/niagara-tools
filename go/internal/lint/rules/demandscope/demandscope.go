// Package demandscope is lint-demand-scope: a control-decision method that reads a process variable in a
// comparison with no demand-shaped input in scope ("pressure without demand"). Port of the awk in
// toolbelt/lint-demand-scope.sh; the wrapper keeps the user-facing contract and its header documents the
// rule and its mutation pins.
//
// It is a per-file lint.Rule: a file's rows depend on that file alone (its methods and its class-level
// fields). Matching keeps gawk's C-locale byte semantics: tolower folds ASCII only.
package demandscope

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts; rows print the check name instead.
const ID = "lint-demand-scope"

// check is the check column of every row, as the bash lint printed it.
const check = "demand-in-scope"

// Spec registers the rule with the argument contract of toolbelt/lint-demand-scope.sh: one optional leading
// --strict, then the root (later arguments ignored), and a root with no .java file is the K20 ERROR row.
// The walk stops on an unreadable directory (scan_files_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{},
		Mode:       lint.Leading,
		Usage:      "usage: lint-demand-scope.sh [--strict] <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Check: check, Msg: "no *.java sources found"},
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// pvs are the process-variable stems in the order the first match names the row.
var pvs = []string{"suction", "pressure", "discharge", "temp", "cv", "coil", "head"}

var (
	// ctrl is rule 1, a control-decision body: it writes a target, a cmd slot, an output setter, or returns
	// a comparison. The awk wrote the last as /\breturn.../, and gawk reads \b as a backspace, not a word
	// boundary, so it is kept as the byte it matched.
	ctrl = regexp.MustCompile(`target[[:space:]]*[+\-]?=|target[+\-][+\-]|cmd\[|setBool\(|\.setValue\(|set[A-Z][A-Za-z0-9]*\(|\x08return[^;]*[<>]`)
	// pvAt is rule 2 per stem: the stem (plus word chars) before " <"/" >", or "<"/">" (plus "=") and a
	// blank before the stem.
	pvAt       = map[string]*regexp.Regexp{}
	staticInit = regexp.MustCompile(`static[[:space:]]*\{`)
	inBoolean  = regexp.MustCompile(`BStatusBoolean[[:space:]]+in[A-Z]`)
	methodName = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*[[:space:]]*$`)
)

func init() {
	for _, pv := range pvs {
		pvAt[pv] = regexp.MustCompile(pv + `[a-z0-9_.]*[[:space:]][<>]|[<>][=]?[[:space:]][a-z0-9_.]*` + pv)
	}
}

// Check is the awk's two passes over the file's lines, each cut at its first "//".
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	lines := make([]string, len(f.Raw))
	for i, ln := range f.Raw {
		lines[i] = cutComment(ln)
	}
	classDemand := false
	depth := 0
	for _, ln := range lines { // pass 1: class-level fields (depth 1, no "(", a ";")
		before := depth
		depth += braces(ln)
		if before == 1 && !strings.Contains(ln, "(") && strings.Contains(ln, ";") && !keyword(ln, "class") &&
			!keyword(ln, "interface") && demandShaped(lower(ln)) {
			classDemand = true
		}
	}
	var out []lint.Finding
	depth = 0
	for i, ln := range lines { // pass 2: methods at class scope
		before := depth
		opens := strings.Count(ln, "{")
		depth += opens - strings.Count(ln, "}")
		if before != 1 || opens == 0 || !strings.Contains(ln, "(") || !strings.Contains(ln, ")") ||
			keyword(ln, "class") || keyword(ln, "interface") || keyword(ln, "enum") || staticInit.MatchString(ln) {
			continue
		}
		body := ln
		for j, cur := i+1, depth; j < len(lines) && cur > 1; j++ {
			body += "\n" + lines[j]
			cur += braces(lines[j])
		}
		if !ctrl.MatchString(body) {
			continue
		}
		pv := readsPV(lower(body))
		if pv == "" || classDemand {
			continue
		}
		params := paramList(ln)
		if demandShaped(lower(params)) || inBoolean.MatchString(params) {
			continue
		}
		out = append(out, lint.Finding{Check: check, Line: i + 1,
			Msg: name(ln) + " reads " + pv + " with no demand-shaped input in scope"})
	}
	return out, nil
}

// cutComment is the awk gsub(/\/\/.*$/, ""): the line up to its first "//", in a literal or not.
func cutComment(ln string) string {
	if k := strings.Index(ln, "//"); k >= 0 {
		return ln[:k]
	}
	return ln
}

// braces is the line's "{" count minus its "}" count.
func braces(ln string) int { return strings.Count(ln, "{") - strings.Count(ln, "}") }

// keyword is the awk's ln ~ /\b<word>\b/, which gawk reads as <word> between two backspaces.
func keyword(ln, word string) bool { return strings.Contains(ln, "\b"+word+"\b") }

// demandShaped reports a lowered name list naming a demand-shaped input: demand*, *call*, enable, *count.
func demandShaped(lo string) bool {
	return strings.Contains(lo, "demand") || strings.Contains(lo, "call") || strings.Contains(lo, "enable") ||
		strings.Contains(lo, "count")
}

// readsPV is the first process-variable stem the lowered body compares, or "".
func readsPV(lo string) string {
	for _, pv := range pvs {
		if pvAt[pv].MatchString(lo) {
			return pv
		}
	}
	return ""
}

// paramList is the text from the line's first "(" to the first ")" after it (the rest when none).
func paramList(ln string) string {
	after := ln[strings.IndexByte(ln, '(')+1:]
	if k := strings.IndexByte(after, ')'); k >= 0 {
		return after[:k]
	}
	return after
}

// name is the last identifier before the line's first "(", or "unknown".
func name(ln string) string {
	head := ln[:strings.IndexByte(ln, '(')]
	m := methodName.FindString(head)
	if m == "" {
		return "unknown"
	}
	return strings.TrimRight(m, " \t\n\v\f\r")
}

// lower is gawk's tolower under LC_ALL=C: ASCII letters only, every other byte unchanged.
func lower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}
