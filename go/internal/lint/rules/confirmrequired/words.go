// Name, flag and waiver helpers of the lint-confirm-required rule.
package confirmrequired

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

// isDestructive: a whole word of name, split at its capitals, digits and underscores, is on the list.
func isDestructive(name string) bool {
	for _, w := range words(name) {
		if destructive[w] {
			return true
		}
	}
	return false
}

// words splits a camelCase or snake_case name into lower-case words: faultReset -> fault, reset;
// URLReset -> url, reset.
func words(name string) []string {
	rs := []rune(name)
	var out []string
	cur := []rune{}
	flush := func() {
		if len(cur) > 0 {
			out = append(out, strings.ToLower(string(cur)))
			cur = cur[:0]
		}
	}
	for i, r := range rs {
		switch {
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			flush()
			continue
		case unicode.IsUpper(r) && i > 0 && (unicode.IsLower(rs[i-1]) || unicode.IsDigit(rs[i-1]) ||
			(unicode.IsUpper(rs[i-1]) && i+1 < len(rs) && unicode.IsLower(rs[i+1]))):
			flush()
		case unicode.IsDigit(r) && i > 0 && !unicode.IsDigit(rs[i-1]):
			flush()
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

// hasConfirm: the flags value names CONFIRM_REQUIRED (Flags.CONFIRM_REQUIRED, qualified or statically
// imported) or holds an integer literal with the 0x80 bit.
func hasConfirm(flags []javasrc.Token) bool {
	for _, t := range flags {
		switch t.Kind {
		case javasrc.TokIdent:
			if t.Text == "CONFIRM_REQUIRED" {
				return true
			}
		case javasrc.TokNumber:
			s := strings.TrimRight(strings.ReplaceAll(t.Text, "_", ""), "lL")
			if v, err := strconv.ParseInt(s, 0, 64); err == nil && v&confirmRequired != 0 {
				return true
			}
		}
	}
	return false
}

// waiver reads the marker for the annotation on 0-based line i: the // comment of that line, else a // comment
// line right above. "reason" (a marker with text after it), "bare" (a marker with nothing after it) or "".
func waiver(raw []string, i int) string {
	if st := state(lineComment(raw[i])); st != "" {
		return st
	}
	if i > 0 && commentLine.MatchString(raw[i-1]) {
		return state(lineComment(raw[i-1]))
	}
	return ""
}

func lineComment(ln string) string {
	if at := javasrc.LineCommentAt(ln); at >= 0 {
		return ln[at:]
	}
	return ""
}

func state(text string) string {
	switch {
	case withReason.MatchString(text):
		return "reason"
	case marker.MatchString(text):
		return "bare"
	}
	return ""
}
