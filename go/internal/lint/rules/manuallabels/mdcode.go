package manuallabels

import "strings"

// markdownCode is a Markdown manual with its code blanked, lines kept (truth fix 2026-10-24: a quoted
// string in code is a JSON key or a shell argument, never a UI label): every line of a fenced block whose
// opening fence names a language (```json, ~~~ bash; indented up to 3 spaces, closed by a fence of the same
// character at least as long, or the end) and each inline code span (a backtick run closed by the next run
// of the same length on its line). A bare fence is kept as text: the manuals draw their diagrams in one,
// and a diagram quotes real labels.
func markdownCode(text string) string {
	lines := strings.Split(text, "\n")
	fence, code := "", false
	for i, ln := range lines {
		t := strings.TrimLeft(ln, " ")
		inset := len(ln) - len(t) // CommonMark: a fence is indented by at most 3 spaces
		switch {
		case fence != "":
			if inset <= 3 && strings.HasPrefix(t, fence) && strings.Trim(t, fence[:1]+" \t") == "" {
				fence = ""
			}
			if code {
				lines[i] = ""
			}
		case inset <= 3 && run(t) >= 3 && (t[0] == '`' || t[0] == '~'):
			fence = t[:run(t)]
			code = strings.TrimSpace(t[len(fence):]) != ""
			if code {
				lines[i] = ""
			}
		default:
			lines[i] = codeSpans(ln)
		}
	}
	return strings.Join(lines, "\n")
}

// run is the length of the run of s's first byte at its start.
func run(s string) int {
	n := 0
	for n < len(s) && s[n] == s[0] {
		n++
	}
	return n
}

// codeSpans replaces each inline code span of a line by one space.
func codeSpans(ln string) string {
	var b strings.Builder
	for i := strings.IndexByte(ln, '`'); i >= 0; i = strings.IndexByte(ln, '`') {
		n := run(ln[i:])
		end := -1
		for j := i + n; j < len(ln); {
			k := strings.IndexByte(ln[j:], '`')
			if k < 0 {
				break
			}
			if m := run(ln[j+k:]); m == n {
				end = j + k + m
				break
			} else {
				j += k + m
			}
		}
		if end < 0 { // no closing run: the backticks are literal
			b.WriteString(ln[:i+n])
		} else {
			b.WriteString(ln[:i] + " ")
		}
		ln = ln[max(end, i+n):]
	}
	return b.String() + ln
}
