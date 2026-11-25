package subscribeleakjs

// The waiver comment of one site: `// lint-subscribe-js: <reason>`.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

const marker = "lint-subscribe-js:"

// reasonAt reads the // comment of 1-based line n: marked is a waiver marker in it, reason its text after the colon,
// commentOnly a line that holds nothing but the comment.
func reasonAt(raw []string, n int) (marked bool, reason string, commentOnly bool) {
	if n < 1 || n > len(raw) {
		return false, "", false
	}
	ln := raw[n-1]
	at := jssrc.LineCommentAt(ln)
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
