package readtools

// The BFormat tokenizer of n4_alarm_text_check, a port of javax.baja.util.BFormat.doParse
// (organized/docSource/docSource-doc/extracted/baja/javax/baja/util/BFormat.java:323-421, the
// same in the vineflower copy). niagara-go internal/bognav/alarmtext_format.go (bog-nav alarm-text)
// holds the same functions: the offline and the live check must agree.

import "fmt"

// badFormatHint is the advice of a bad-format finding.
const badFormatHint = `BFormat cannot parse the text (a % never closes) and shows it as "ERROR <text>": write %% for a literal percent`

// textFinding is one finding of one text: its rule and its token ("" for bad-format).
type textFinding struct{ rule, token string }

// textFindings are the findings of one text: one bad-format when BFormat cannot parse it (no token
// of it is ever evaluated), else one parent-path per unresolved token.
func textFindings(text string) []textFinding {
	if _, ok := formatScripts(text); !ok {
		return []textFinding{{ruleBadFormat, ""}}
	}
	var out []textFinding
	for _, tok := range unresolvedAlarmTokens(text) {
		out = append(out, textFinding{ruleParentPath, tok})
	}
	return out
}

// alarmTextCounts is the summary's count list; bad-format is named only when a text has it.
func alarmTextCounts(parents, empties, bad int) string {
	s := fmt.Sprintf("%d parent-path, %d empty-tonormal", parents, empties)
	if bad > 0 {
		s += fmt.Sprintf(", %d bad-format", bad)
	}
	return s
}

// formatScripts are the scripts of a BFormat pattern in order, each the text between its two
// percent signs. Outside a script %% is a literal percent (BFormat.java:337-339); any other %
// opens a script that ends at the next % outside parentheses (:350). A % that never closes, a
// lone % anywhere included, makes doParse read past the end and BFormat renders the whole text as
// "ERROR <text>" (:304-315): ok is false and no script is ever evaluated.
func formatScripts(text string) (scripts []string, ok bool) {
	for i := 0; i < len(text); i++ {
		if text[i] != '%' {
			continue
		}
		if i+1 == len(text) {
			return nil, false
		}
		if text[i+1] == '%' {
			i++
			continue
		}
		end := scriptEnd(text, i+1)
		if end < 0 {
			return nil, false
		}
		scripts = append(scripts, text[i+1:end])
		i = end
	}
	return scripts, true
}

// scriptEnd is the index of the % that closes the script starting at start, -1 when none does.
// Parentheses nest, so a % or ? inside them is script text; the first ? outside them starts the
// error fallback, still inside the same pair, and restarts the parenthesis count (:372-380).
func scriptEnd(text string, start int) int {
	depth, fallback := 0, false
	for i := start; i < len(text); i++ {
		c := text[i]
		if depth <= 0 {
			if c == '%' {
				return i
			}
			if c == '?' && !fallback {
				fallback, depth = true, 0
				continue
			}
		}
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		}
	}
	return -1
}
