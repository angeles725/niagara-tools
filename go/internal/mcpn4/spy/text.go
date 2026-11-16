package spy

import "strings"

// oneLine is s with every run of white space as one space, trimmed.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// lines is s with each line as oneLine and the empty lines dropped.
func lines(s string) string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = oneLine(l); l != "" {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// clip is s cut at max runes and whether it was cut.
func clip(s string, max int) (string, bool) {
	r := []rune(s)
	if len(r) <= max {
		return s, false
	}
	return string(r[:max]), true
}
