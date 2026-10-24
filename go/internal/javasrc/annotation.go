package javasrc

import (
	"regexp"
	"strings"
)

// annotationName is an annotation's name and the blanks after it, at the "@" that starts it.
var annotationName = regexp.MustCompile(`^@[A-Za-z_][A-Za-z0-9_.]*[[:space:]]*`)

// AnnotationSkip skips annotations in blanked code (see BlankLiterals) read char by char: an annotation
// is its name and, when a "(" follows the name, its balanced argument list, on one line or several.
// The zero value is outside any annotation.
type AnnotationSkip struct {
	open int // parentheses open in the argument list being skipped (0 outside one)
}

// Open reports whether an annotation's argument list is still being skipped.
func (a *AnnotationSkip) Open() bool { return a.open > 0 }

// Step reads code[k]: n is how many bytes from k the annotation consumes (0 when code[k] neither starts
// an annotation nor lies inside an open argument list) and ended reports that the annotation ends with
// them, where a reader that replaces each annotation by one blank writes it.
func (a *AnnotationSkip) Step(code string, k int) (n int, ended bool) {
	c := code[k]
	switch {
	case a.open > 0:
		if c == '(' {
			a.open++
		} else if c == ')' {
			a.open--
		}
		return 1, a.open == 0
	case c == '@':
		m := len(annotationName.FindString(code[k:]))
		if m == 0 {
			return 0, false
		}
		if k+m < len(code) && code[k+m] == '(' {
			a.open = 1
			return m + 1, false
		}
		return m, true
	}
	return 0, false
}

// NiagaraProperties calls fn(text, line) for every @NiagaraProperty annotation in lines (raw or Strip
// output; the caller picks the view): from the line holding "@NiagaraProperty" on, each line prefixed with
// " " and joined, until a "(" was seen and the parentheses balance (ParenDepth <= 0). line is the 1-based
// line of the opening. A one-line annotation closes on its own line, so an annotation on the next line is
// read too. An annotation still open at the end of lines is dropped. Parentheses are counted as text,
// inside literals too (the lints this serves always read them so).
func NiagaraProperties(lines []string, fn func(text string, line int)) {
	scan := PropertyScan{NeedParen: true}
	for i, ln := range lines {
		if text, line, ok := scan.Line(ln, i+1); ok {
			fn(text, line)
		}
	}
}

// PropertyScan is the @NiagaraProperty accumulator NiagaraProperties runs, fed one line at a time so a
// caller can carry an open annotation from one file into the next (one awk over several files does).
// NeedParen closes an annotation only once a "(" was seen; without it a line that holds
// "@NiagaraProperty" and no unbalanced "(" closes at once (the awk scan of slot-coverage.sh).
type PropertyScan struct {
	NeedParen bool
	open      bool
	buf       string
	line      int
}

// Line reads ln (line n): when ln closes an annotation it returns the joined text and its opening line.
func (p *PropertyScan) Line(ln string, n int) (text string, line int, ok bool) {
	if !p.open && strings.Contains(ln, "@NiagaraProperty") {
		p.open, p.buf, p.line = true, "", n
	}
	if !p.open {
		return "", 0, false
	}
	p.buf += " " + ln
	if ParenDepth(p.buf) > 0 || (p.NeedParen && !strings.Contains(p.buf, "(")) {
		return "", 0, false
	}
	text, line = p.buf, p.line
	p.open, p.buf, p.line = false, "", 0
	return text, line, true
}

// ParenDepth is the count of "(" in s minus the count of ")".
func ParenDepth(s string) int { return strings.Count(s, "(") - strings.Count(s, ")") }
