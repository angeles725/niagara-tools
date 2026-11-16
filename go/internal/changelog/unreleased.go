package changelog

import (
	"regexp"
	"strconv"
	"strings"
)

var hunk = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// addsUnreleased reports whether the range's CHANGELOG.md diff adds a non-blank line strictly inside the
// `## [Unreleased]` section of the head version.
func addsUnreleased(rng, head string, git gitRunner) (bool, error) {
	cur, err := git("show", head+":CHANGELOG.md")
	if err != nil {
		return false, err
	}
	start, end, ok := UnreleasedBounds(strings.Split(cur, "\n"))
	if !ok {
		return false, nil
	}
	patch, err := git("diff", "-U0", "--no-renames", rng, "--", "CHANGELOG.md")
	if err != nil {
		return false, err
	}
	line := 0 // 1-based line number in the head version of the next '+' or ' ' line
	for _, l := range strings.Split(patch, "\n") {
		if m := hunk.FindStringSubmatch(l); m != nil {
			line, _ = strconv.Atoi(m[1])
			continue
		}
		if line == 0 || strings.HasPrefix(l, "+++") || !strings.HasPrefix(l, "+") {
			continue
		}
		if idx := line - 1; idx > start && idx < end && strings.TrimSpace(l[1:]) != "" {
			return true, nil
		}
		line++
	}
	return false, nil
}
