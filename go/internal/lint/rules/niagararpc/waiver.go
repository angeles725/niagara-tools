package niagararpc

// The reviewed-marker waiver of one RPC.

import (
	"regexp"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
)

var (
	marker   = regexp.MustCompile(`lint-niagara-rpc:[ \t]*reviewed`)
	reasoned = regexp.MustCompile(`^[ \t]+\S`)
)

// waiver reads the // comments of lines lo-1 .. hi (1-based): waived is a marker with a reason; bare is the line of
// the first marker with none (0 when there is none).
func waiver(raw []string, annLine, hi int) (waived bool, bare int) {
	for l := max(annLine-1, 1); l <= hi && l <= len(raw); l++ {
		at := javasrc.LineCommentAt(raw[l-1])
		if at < 0 {
			continue
		}
		loc := marker.FindStringIndex(raw[l-1][at:])
		if loc == nil {
			continue
		}
		if reasoned.MatchString(raw[l-1][at+loc[1]:]) {
			waived = true
		} else if bare == 0 {
			bare = l
		}
	}
	return waived, bare
}
