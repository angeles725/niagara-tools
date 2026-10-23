// Package linescan holds the line helpers shared by the rules ported from the line-matching bash lints
// (awk_lint_each and awk_lint_strip_run in toolbelt/lib/awk-lint.sh): the row text they print and the
// line view they match. A rule that matches lines uses these, never a private copy.
package linescan

import "strings"

// space is awk's and sed's [[:space:]] in the C locale.
const space = " \t\n\v\f\r"

// Cut is ln up to its first "//", as the bash lints cut a line comment (awk `index(code, "//")`,
// sed 's://.*::'): string-unaware, so a "//" inside a literal also cuts the line.
func Cut(ln string) string {
	if i := strings.Index(ln, "//"); i >= 0 {
		return ln[:i]
	}
	return ln
}

// Source is the row text of an awk_lint_each lint: the raw line with its leading space removed
// (awk `sub(/^[[:space:]]+/, "", t)`); trailing space, "\r" included, is kept.
func Source(ln string) string { return strings.TrimLeft(ln, space) }

// Trimmed is the row text of an awk_lint_strip_run lint: the raw line with leading and trailing space
// removed (sed 's/^[[:space:]]*//; s/[[:space:]]*$//').
func Trimmed(ln string) string { return strings.Trim(ln, space) }
