// Waiver scope of the lint-wb-usability rule.
package wbusability

import (
	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// floor is the last 0-based line a waiver for c must not reach: the line of the closing brace of the nearest class
// that ends before c, or the line of the opening brace of the class that encloses c. A waiver above another class,
// or inside its body, is that class's business. -1 when nothing bounds c.
func floor(toks []javasrc.Token, classes []javasrc.Class, c javasrc.Class) int {
	at := -1
	for _, d := range classes {
		switch {
		case d.At == c.At:
		case d.Body[1] < c.At && d.Body[1] < len(toks):
			at = max(at, toks[d.Body[1]].Line)
		case d.At < c.At && c.At < d.Body[1] && d.Body[0] > 0:
			at = max(at, toks[d.Body[0]-1].Line)
		}
	}
	return at
}

// waived: a comment with the waiver and a reason stands on class line (0-based) or in the waiverWindow lines above
// it, but never at or above floor (the end of the previous class). The marker counts only where Strip blanked it, so
// a string literal holding the same text is not a waiver.
func waived(f *javasrc.File, line, floor int) bool {
	stripped := f.Stripped()
	for i := line; i >= 0 && i >= line-waiverWindow && (i > floor || i == line); i-- {
		if waiver.MatchString(f.Raw[i]) && !waiver.MatchString(stripped[i]) {
			return true
		}
	}
	return false
}
