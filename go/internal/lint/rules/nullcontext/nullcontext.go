// Package nullcontext is the lint-null-context-write rule: component writes and bql: ORD resolutions
// made with no caller Context. Port of the awk in toolbelt/lint-null-context-write.sh (the wrapper keeps
// the user-facing contract and its header documents the rule, its mutation pins and the declared
// differences: a write is judged on its own argument list, across lines).
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
	// The start of a write call; its argument list is read to its balanced ")" across lines.
	write = regexp.MustCompile(`\.(set|setInt|setBoolean|setDouble|invoke)[[:space:]]*\(`)
	// A null Context argument: null, or null cast to Context ((Context) null, (javax.baja.sys.Context)null).
	nullArg = regexp.MustCompile(`^(\([[:space:]]*([A-Za-z_][A-Za-z0-9_]*[[:space:]]*\.[[:space:]]*)*Context[[:space:]]*\)[[:space:]]*)?null$`)
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
	var own *seeds // built on the first null-Context write: most files have none
	for i, ln := range code {
		for _, loc := range write.FindAllStringIndex(ln, -1) {
			args, ok := nullLastArg(code, i, loc[1])
			if !ok {
				continue
			}
			if own == nil {
				own = newSeeds(code)
			}
			if !own.internal(i, ln[:loc[0]], args[:len(args)-1]) {
				report(i, "null-Context write: ") // one row per line, at the line of the call
				break
			}
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

// nullLastArg reports whether the call whose argument list opens at code[i][at:] has two or more
// arguments and its last one is a null Context, and returns the arguments. The list is read to its balanced
// ")" across lines (the whole statement, not one line); an argument list still open at the end of the file
// is not judged.
func nullLastArg(code []string, i, at int) ([]string, bool) {
	var args []string
	var cur strings.Builder
	depth := 0
	for ln, j := code[i][at:], i; ; {
		for k := 0; k < len(ln); k++ {
			switch c := ln[k]; {
			case c == '(' || c == '[' || c == '{':
				depth++
			case (c == ')' || c == ']' || c == '}') && depth > 0:
				depth--
			case c == ')':
				args = append(args, cur.String())
				return args, len(args) >= 2 && nullArg.MatchString(strings.Trim(args[len(args)-1], space))
			case c == ',' && depth == 0:
				args = append(args, cur.String())
				cur.Reset()
				continue
			}
			cur.WriteByte(ln[k])
		}
		if j++; j == len(code) {
			return nil, false
		}
		ln = code[j]
		cur.WriteByte(' ')
	}
}
