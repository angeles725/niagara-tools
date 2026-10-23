package javasrc

import "regexp"

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
