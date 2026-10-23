// Package nullcontext is the lint-null-context-write rule: component writes and bql: ORD resolutions
// made with no caller Context. Port of the awk in toolbelt/lint-null-context-write.sh (the wrapper keeps
// the user-facing contract and its header documents the rule and its mutation pins).
package nullcontext

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

const id = "lint-null-context-write"

// Spec registers the rule with the argument contract of toolbelt/lint-null-context-write.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-null-context-write.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_run, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return id }
func (rule) Severity() lint.Severity { return lint.Warn }

var (
	// A write whose last argument is null, on one blanked code line.
	write = regexp.MustCompile(`\.(set|setInt|setBoolean|setDouble|invoke)\(.*,[[:space:]]*null[[:space:]]*\)`)
	// The blanked ORD literal, an optional paren-free concatenation, make's ")", then the call on that ORD.
	ordCall = regexp.MustCompile(`^""[^()]*\)[[:space:]]*\.(get|resolve)\(`)
	noArgs  = regexp.MustCompile(`^[[:space:]]*\)`)
	oneArg  = regexp.MustCompile(`^([^(),]|\([^()]*\))+\)`)
	// The call's own argument list (up to its balanced ")") ends in a null argument.
	nullLast = regexp.MustCompile(`,[[:space:]]*null[[:space:]]*$`)
)

const space = " \t\n\v\f\r" // awk's [[:space:]] in the C locale

// Check never fails: every file it is given is scanned line by line, as the awk did.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	// bql[i] is the 1-based column in code[i] of the opening quote of the first "..." literal on line i
	// whose text holds "bql:" (0 when none).
	st := f.Stripped()
	bql := make([]int, len(st))
	code, _ := javasrc.BlankLiterals(st, func(l javasrc.Literal) {
		if l.Quote == '"' && bql[l.Line] == 0 && strings.Contains(l.Text, "bql:") {
			bql[l.Line] = l.Col
		}
	})
	var out []lint.Finding
	report := func(i int, msg string) {
		out = append(out, lint.Finding{Line: i + 1, Msg: msg + strings.TrimLeft(f.Raw[i], space)})
	}
	for i, ln := range code {
		if write.MatchString(ln) {
			report(i, "null-Context write: ")
		}
	}
	// -wb parts are client code (the session carries the user): skipped by the bql rule only.
	if strings.Contains(f.Path, "-wb/") {
		return out, nil
	}
	for i := range code {
		if bql[i] == 0 {
			continue
		}
		// The statement window: from the bql literal to the first ";" or "{", lines joined by a space.
		w := code[i][bql[i]-1:]
		for k := i; !strings.ContainsAny(w, ";{") && k < len(code)-1; {
			k++
			w += " " + code[k]
		}
		if j := strings.IndexAny(w, ";{"); j >= 0 {
			w = w[:j]
		}
		loc := ordCall.FindStringIndex(w)
		if loc == nil {
			continue
		}
		if args := w[loc[1]:]; noArgs.MatchString(args) || oneArg.MatchString(args) || nullContext(args) {
			report(i, "bql: ORD resolved without the caller's Context (pass get(base, cx)): ")
		}
	}
	return out, nil
}

// nullContext reports whether the argument list that args opens (the text after "get(") ends in a
// null argument before its own closing parenthesis. A null after that parenthesis belongs to another
// call (issue #293).
func nullContext(args string) bool {
	depth := 0
	for j := 0; j < len(args); j++ {
		switch args[j] {
		case '(':
			depth++
		case ')':
			if depth == 0 {
				return nullLast.MatchString(args[:j])
			}
			depth--
		}
	}
	return false
}
