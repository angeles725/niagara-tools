package modulemap

import (
	"path"
	"strings"
)

// matchGlob matches rel against pat segment by segment: "**" spans any number of segments (none included),
// every other segment is a path.Match pattern (so * and ? stay inside one segment). A malformed pattern
// matches nothing.
func matchGlob(pat, rel string) bool {
	return matchSegs(strings.Split(pat, "/"), strings.Split(rel, "/"))
}

func matchSegs(p, r []string) bool {
	if len(p) == 0 {
		return len(r) == 0
	}
	if p[0] == "**" {
		for i := 0; i <= len(r); i++ {
			if matchSegs(p[1:], r[i:]) {
				return true
			}
		}
		return false
	}
	if len(r) == 0 {
		return false
	}
	ok, err := path.Match(p[0], r[0])
	return err == nil && ok && matchSegs(p[1:], r[1:])
}

// MatchGlob reports whether the map entry pat (a "**"/"*" glob of the Files or Tests cell) matches rel.
func MatchGlob(pat, rel string) bool { return matchGlob(path.Clean(pat), rel) }
