package subscribeleakjs

// The waiver comment of one site: `// lint-subscribe-js: <reason>`.

import "strings"

const marker = "lint-subscribe-js:"

// reasonAt reads the // comment of 1-based line n: marked is a waiver marker in it, reason its text after the colon,
// commentOnly a line that holds nothing but the comment.
func reasonAt(raw []string, n int) (marked bool, reason string, commentOnly bool) {
	if n < 1 || n > len(raw) {
		return false, "", false
	}
	ln := raw[n-1]
	at := commentAt(ln)
	if at < 0 {
		return false, "", false
	}
	i := strings.Index(ln[at:], marker)
	if i < 0 {
		return false, "", false
	}
	return true, strings.TrimSpace(ln[at+i+len(marker):]), strings.TrimSpace(ln[:at]) == ""
}

// waived: the site on 1-based line n has a marker with a reason on its own line, or on the line above when that
// line holds only the comment.
func waived(raw []string, n int) bool {
	if m, reason, _ := reasonAt(raw, n); m && reason != "" {
		return true
	}
	m, reason, only := reasonAt(raw, n-1)
	return m && reason != "" && only
}

// bare lists the 1-based lines whose waiver marker has no reason.
func bare(raw []string) []int {
	var out []int
	for n := range raw {
		if m, reason, _ := reasonAt(raw, n+1); m && reason == "" {
			out = append(out, n+1)
		}
	}
	return out
}

// commentAt is the index of the // that starts a comment outside a '...', "..." or `...` literal of one line (a
// literal open at the line end is read to the end), or -1; an inline /* ... */ is skipped and a /* that does not
// close ends the search. Same reading as bqlunbounded's jsCommentAt (a quote inside a regex literal is not read).
func commentAt(s string) int {
	var q byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if q != 0 {
			if c == '\\' {
				i++
			} else if c == q {
				q = 0
			}
			continue
		}
		switch {
		case c == '"' || c == '\'' || c == '`':
			q = c
		case c == '/' && i+1 < len(s) && s[i+1] == '/':
			return i
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			e := strings.Index(s[i+2:], "*/")
			if e < 0 {
				return -1
			}
			i += 2 + e + 1
		}
	}
	return -1
}
