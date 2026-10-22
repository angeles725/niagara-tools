// Package driverscaffold is the lint-driver-scaffold rule: NDriver wizard stubs left in shipped driver
// sources (a doPing that only calls pingOk() FAILs; the wizard's other placeholders WARN). Port of the awk
// in toolbelt/lint-driver-scaffold.sh (the wrapper keeps the user-facing contract and its header
// documents the rule and its mutation pins), with the edge cases of issue #282 fixed.
package driverscaffold

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the rule name printed in every row.
const ID = "lint-driver-scaffold"

// Spec registers the rule with the argument contract of toolbelt/lint-driver-scaffold.sh.
func Spec() lint.Spec {
	return lint.Spec{
		Rule:   rule{},
		Mode:   lint.Raw,
		Usage:  "usage: lint-driver-scaffold.sh <src-root>",
		Select: lint.Selection{Names: []string{"*.java"}},
		// Reason: bash used awk_lint_list, which scans the partial list (env line, then rows, exit 3).
		ContinueOnWalkError: true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

// markers are the wizard TODOs that WARN wherever they appear, comments included.
var markers = []string{"TODO add poll support", "TODO - convert linkMessage", "TODO add code to detect beginning and end of message"}

const space = " \t\n\v\f\r" // awk's [[:space:]] in the C locale

// Check never fails: a file whose boundaries cannot be trusted gets one WARN row instead.
func (rule) Check(f *javasrc.File) ([]lint.Finding, error) {
	code, unterm := blank(f.Stripped())
	s := &scanner{path: f.Path}
	for i, raw := range f.Raw {
		for _, m := range markers {
			if strings.Contains(raw, m) {
				s.row(lint.Warn, i+1, "wizard stub: "+strings.TrimLeft(raw, space))
				break
			}
		}
		s.line(i+1, raw, code[i])
	}
	why := ""
	switch {
	case f.End() == javasrc.OpenComment:
		why = "ends inside a block comment"
	case f.End() == javasrc.OpenTextBlock:
		why = "ends inside a text block"
	case unterm:
		why = "an unterminated string or char literal"
	case s.ann > 0:
		why = "an annotation's argument list does not close"
	case s.depth != 0 || s.negative:
		why = "braces do not balance"
	}
	// An open comment, literal or annotation is named first: it is why the braces after it look unbalanced.
	if why != "" {
		s.row(lint.Warn, len(f.Raw), "method boundaries not trusted ("+why+"): a stub in this file may be missed")
	}
	return s.out, nil
}

// blank returns the stripped lines with the contents of "...", '...' and """...""" literals removed
// (quotes kept, \ escapes honoured, text-block state carried across lines), so a { } ; ( @ or
// "pingOk();" inside one is never code, and reports whether a "..." or '...' was still open at a line end.
func blank(src []string) (dst []string, unterm bool) {
	dst = make([]string, len(src))
	inText := false
	var b strings.Builder
	for i, ln := range src {
		b.Reset()
		var quote byte
		for j := 0; j < len(ln); j++ {
			c := ln[j]
			switch {
			case inText || quote != 0:
				if c == '\\' {
					j++
				} else if inText && strings.HasPrefix(ln[j:], `"""`) {
					b.WriteString(`"""`)
					inText = false
					j += 2
				} else if !inText && c == quote {
					b.WriteByte(c)
					quote = 0
				}
			case strings.HasPrefix(ln[j:], `"""`):
				b.WriteString(`"""`)
				inText = true
				j += 2
			default:
				if c == '"' || c == '\'' {
					quote = c
				}
				b.WriteByte(c)
			}
		}
		unterm = unterm || quote != 0
		dst[i] = b.String()
	}
	return dst, unterm
}
