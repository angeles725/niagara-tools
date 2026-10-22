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

func (rule) Check(f *javasrc.File) []lint.Finding {
	code, bql := blank(f.Stripped())
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
		return out
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
	return out
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

// blank returns the stripped lines with the contents of "...", '...' and """...""" literals removed
// (quotes kept, \ escapes honoured, text-block state carried across lines; a "..." or '...' literal
// ends at its line end) and, per line, the 1-based column in the blanked line of the opening quote of
// the first "..." literal on that line whose text holds "bql:" (0 when none).
func blank(src []string) (dst []string, bql []int) {
	dst, bql = make([]string, len(src)), make([]int, len(src))
	inText := false
	var b, lit strings.Builder
	for i, ln := range src {
		b.Reset()
		var quote byte
		at := 0
		for j := 0; j < len(ln); j++ {
			c := ln[j]
			switch {
			case inText:
				if c == '\\' {
					j++
				} else if strings.HasPrefix(ln[j:], `"""`) {
					b.WriteString(`"""`)
					inText = false
					j += 2
				}
			case quote != 0:
				switch c {
				case '\\':
					lit.WriteString(ln[j:min(j+2, len(ln))])
					j++
				case quote:
					b.WriteByte(quote)
					if quote == '"' && bql[i] == 0 && strings.Contains(lit.String(), "bql:") {
						bql[i] = at
					}
					quote = 0
				default:
					lit.WriteByte(c)
				}
			case strings.HasPrefix(ln[j:], `"""`):
				b.WriteString(`"""`)
				inText = true
				j += 2
			default:
				if c == '"' || c == '\'' {
					quote = c
					lit.Reset()
					at = b.Len() + 1
				}
				b.WriteByte(c)
			}
		}
		dst[i] = b.String()
	}
	return dst, bql
}
