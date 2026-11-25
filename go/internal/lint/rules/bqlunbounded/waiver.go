package bqlunbounded

// The waiver comment of one query: `// lint-bql-unbounded: <reason>`.

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/jssrc"
)

const marker = "lint-bql-unbounded:"

// reasonAt reads the // comment of 1-based line n: marked is a waiver marker in it, reason its text after the colon.
func reasonAt(raw []string, n int, js bool) (marked bool, reason string, commentOnly bool) {
	if n < 1 || n > len(raw) {
		return false, "", false
	}
	ln := raw[n-1]
	at := commentAt(ln, js)
	if at < 0 {
		return false, "", false
	}
	i := strings.Index(ln[at:], marker)
	if i < 0 {
		return false, "", false
	}
	return true, strings.TrimSpace(ln[at+i+len(marker):]), strings.TrimSpace(ln[:at]) == ""
}

// waived: the query on 1-based line n has a marker with a reason on its own line, or on the line above when that
// line holds only the comment.
func waived(raw []string, n int, js bool) bool {
	if m, reason, _ := reasonAt(raw, n, js); m && reason != "" {
		return true
	}
	m, reason, only := reasonAt(raw, n-1, js)
	return m && reason != "" && only
}

// bare lists the 1-based lines whose waiver marker has no reason.
func bare(raw []string, js bool) []int {
	var out []int
	for n := range raw {
		if m, reason, _ := reasonAt(raw, n+1, js); m && reason == "" {
			out = append(out, n+1)
		}
	}
	return out
}

// commentAt is the byte index where the // comment of a line starts, or -1. Java reads it with javasrc.LineCommentAt,
// which knows "..." and '...' only; a JavaScript line also has `...` templates, whose apostrophe would open a
// quote that never closes there, so a .js line is read by jssrc.LineCommentAt.
func commentAt(ln string, js bool) int {
	if js {
		return jssrc.LineCommentAt(ln)
	}
	return javasrc.LineCommentAt(ln)
}
