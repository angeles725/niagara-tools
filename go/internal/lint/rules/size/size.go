// Package size is the lint-size rule: an advisory size smell. A file whose hand-written lines (non-blank
// after comments are blanked, outside every slotomatic region) exceed --max-class WARNs once, named after
// its first class, interface or enum declaration, and every method of a pure class (no @NiagaraType, no
// `extends B<Upper>`) spanning more than --max-method lines WARNs; a slotomatic region that is unclosed,
// nested or a stray END is its own WARN. Port of the awk in toolbelt/lint-size.sh (the wrapper keeps the user-facing contract
// and its header documents the rules and their mutation pins). A file under a module map (lint.ModuleMode) gets
// the wider method rule of mapped (every class, wrapped signatures) and the adapter-branches WARN; its class and
// method rows FAIL in a strict module. A file with no map keeps the rules above byte for byte.
package size

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// ID is the lint name printed in every row.
const ID = "lint-size"

const (
	maxClass  = "--max-class"
	maxMethod = "--max-method"
	beginMark = "BEGIN BAJA AUTO GENERATED CODE"
	endMark   = "END BAJA AUTO GENERATED CODE"
	space     = " \t\n\v\f\r" // awk's [[:space:]] in the C locale
)

// Spec registers the rule with the argument contract of toolbelt/lint-size.sh: [--strict] [--max-class <n>]
// [--max-method <n>] [--] and exactly one root (One).
func Spec() lint.Spec {
	return lint.Spec{
		Rule:       rule{maxClass: 800, maxMethod: 80},
		Mode:       lint.One,
		Usage:      "usage: lint-size.sh [--strict] [--max-class <n>] [--max-method <n>] <src-root>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no files to scan (K20: wrong path or empty scaffold)"},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
		IntFlags:            []lint.IntFlag{{Name: maxClass, Default: 800}, {Name: maxMethod, Default: 80}},
		Configure: func(v map[string]int) lint.Rule {
			return rule{maxClass: v[maxClass], maxMethod: v[maxMethod]}
		},
	}
}

type rule struct{ maxClass, maxMethod int }

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

var (
	extendsBaja = rx(`extends[[:space:]]+B[A-Z][A-Za-z0-9_]*`)
	classDecl   = rx(`(^|[[:space:]])(class|interface|enum)[[:space:]]+[A-Za-z_][A-Za-z0-9_]*`)
	classHead   = rx(`^[[:space:]]*(class|interface|enum)[[:space:]]+`)
)

// Check prints the region rows, then the class row, then the method rows of a pure class.
func (r rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code := f.Stripped()
	gen, out := regions(f.Raw, code)
	blanked := linescan.Code(f)
	hand, name, line, baja := 0, "", 0, false
	for i, c := range code {
		k := blanked[i] // the Baja test and the class name are read in code, never in a literal's text
		baja = baja || strings.Contains(k, "@NiagaraType") || extendsBaja.MatchString(k)
		if m := classDecl.FindString(k); name == "" && m != "" {
			name, line = m[len(classHead.FindString(m)):], i+1
		}
		if !gen[i] && strings.Trim(c, space) != "" {
			hand++
		}
	}
	if name == "" {
		name, line = "?", 1
	}
	mode := lint.ModuleMode(f.Path)
	sev := lint.Unset // the rule's WARN; a strict module's size rows FAIL
	if mode == lint.ModularityStrict {
		sev = lint.Fail
	}
	if hand > r.maxClass {
		out = append(out, lint.Finding{Severity: sev, Line: line, Msg: fmt.Sprintf("class %s has %d hand-written lines (> %d, slotomatic"+
			" regions excluded) -- split by concern into pure classes / child components (types/logic.md § Composition)",
			name, hand, r.maxClass)})
	}
	if mode != lint.ModularityNone {
		return append(out, r.mapped(f, code, gen, sev, name, line, baja)...), nil
	}
	if baja {
		return out, nil
	}
	for _, m := range f.Methods() {
		if span := m.End - m.Start + 1; span > r.maxMethod {
			out = append(out, lint.Finding{Line: m.Start, Msg: fmt.Sprintf("method %s() in pure class %s spans %d lines"+
				" (> %d) -- split into named phase methods (types/logic.md § Pure-class extraction)", m.Name, name, span, r.maxMethod)})
		}
	}
	return out, nil
}

// regions marks the lines inside a closed slotomatic region and returns the nested-region, stray-end and
// unclosed-region rows; an unclosed or nested region's lines stay hand-written (fail closed). A marker
// is one in a comment: on the raw line and gone from its Strip line (code), so a string literal quoting
// it is not one.
func regions(raw, code []string) ([]bool, []lint.Finding) {
	gen := make([]bool, len(raw))
	var out []lint.Finding
	open, at := false, 0 // at: 0-based line of the open BEGIN
	marker := func(i int, m string) bool { return strings.Contains(raw[i], m) && !strings.Contains(code[i], m) }
	for i := range raw {
		switch isEnd := marker(i, endMark); {
		case marker(i, beginMark):
			if open {
				clear(gen[at:i])
				out = append(out, lint.Finding{Line: i + 1, Msg: fmt.Sprintf("nested-region: %s inside the region opened"+
					" at line %d -- that region is counted as hand-written", beginMark, at+1)})
			}
			open, at, gen[i] = true, i, true
		case isEnd && !open:
			out = append(out, lint.Finding{Line: i + 1, Msg: "stray-end: " + endMark + " with no open region"})
		case open:
			gen[i], open = true, !isEnd
		}
	}
	if open {
		clear(gen[at:])
		out = append(out, lint.Finding{Line: at + 1, Msg: "unclosed-region: " + beginMark +
			" with no END marker -- its lines are counted as hand-written"})
	}
	return gen, out
}

// branchKeyword is a hand-written decision point the adapter budget counts.
var branchKeyword = map[string]bool{"if": true, "case": true, "for": true, "while": true, "catch": true}

// maxAdapterBranches is the adapter decision budget (modularity-gaps Δ5): WARN-only until calibrated.
const maxAdapterBranches = 15

// mapped is the method rule of a module with a MODULE-MAP.md (legacy WARN, strict FAIL): every method and
// constructor of every class, Baja or pure, measured from its name's line to its closing brace (javasrc.Decls,
// so a signature wrapped over several lines counts from its first line); then the adapter decision budget, a
// WARN in both modes: a Baja class that imports a model/ peer and keeps more than 15 hand-written branches.
func (r rule) mapped(f *javasrc.File, code []string, gen []bool, sev lint.Severity, name string, line int, baja bool) []lint.Finding {
	var out []lint.Finding
	toks := javasrc.Tokens(code)
	for _, d := range javasrc.Decls(toks) {
		if span := toks[d.Body[1]].Line - d.Line + 1; span > r.maxMethod {
			out = append(out, lint.Finding{Severity: sev, Line: d.Line + 1, Msg: fmt.Sprintf("method %s() in class %s spans %d"+
				" lines (> %d) -- split into named phase methods (types/logic.md § Pure-class extraction)", d.Name, d.Class, span, r.maxMethod)})
		}
	}
	if !baja || !importsModel(code) {
		return out
	}
	branches := 0
	for _, t := range toks {
		if t.Kind == javasrc.TokIdent && branchKeyword[t.Text] && !gen[t.Line] {
			branches++
		}
	}
	if branches > maxAdapterBranches {
		out = append(out, lint.Finding{Severity: lint.Warn, Line: line, Msg: fmt.Sprintf("adapter-branches: class %s has %d"+
			" hand-written branches (> %d) beside its model/ peer -- move the decisions into the pure class (types/logic.md"+
			" § Pure-class extraction)", name, branches, maxAdapterBranches)})
	}
	return out
}

// modelImport is an import of a type in a model package (the adapter's pure peer).
var modelImport = regexp.MustCompile(`^[[:space:]]*import[[:space:]]+([A-Za-z_][A-Za-z0-9_]*\.)*model\.[A-Za-z_]`)

func importsModel(code []string) bool {
	for _, c := range code {
		if modelImport.MatchString(c) {
			return true
		}
	}
	return false
}
