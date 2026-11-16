package reportmodule

import (
	"regexp"
	"strings"
)

// failRow is a FAIL row without its status.
type failRow struct{ label, check, detail string }

// pathToken is a word of a detail that holds a "/": a path, absolute or not.
var pathToken = regexp.MustCompile("[^\\s()'\"`,;]*/[^\\s()'\"`,;]*")

// rowKey is the baseline key of a FAIL row: `<artifact>  <check>  <detail>` with the parts of the detail that
// move with unrelated edits taken out, so the same finding keeps its key when lines shift, the checkout moves
// or a count changes. Two rows that differ only in those parts share one key: the key names the finding
// (artifact, check, file, reason), not each of its sites.
func rowKey(w failRow) string {
	return strings.TrimRight(w.label+"  "+w.check+"  "+stableDetail(w.detail), " \t")
}

// stableDetail cuts every path of a detail to its file name and masks every number standing alone.
func stableDetail(detail string) string {
	return maskNumbers(pathToken.ReplaceAllStringFunc(detail, baseName))
}

// maskNumbers replaces each run of digits that touches no letter or "_" with "N": a line or column number
// (`A.java:12`), a count (`81 lines`), a size. Digits inside a word (`ES2020`, `hmi1Write`) are part of a name
// and stay.
func maskNumbers(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if !isDigit(s[i]) {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := i
		for j < len(s) && isDigit(s[j]) {
			j++
		}
		if i > 0 && isWordByte(s[i-1]) || j < len(s) && isWordByte(s[j]) {
			b.WriteString(s[i:j])
		} else {
			b.WriteByte('N')
		}
		i = j
	}
	return b.String()
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isWordByte is an ASCII letter, "_" or a byte of a non-ASCII character.
func isWordByte(c byte) bool {
	return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= 0x80
}
