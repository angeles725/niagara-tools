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

// unreleasedState is what a range does to `## [Unreleased]`.
type unreleasedState int

const (
	unreleasedUntouched      unreleasedState = iota // no line added inside it
	unreleasedInRelease                             // lines added by a release (the fold), an accepted entry
	unreleasedOutsideRelease                        // lines added outside a release: refused
)

func unreleasedVerdict(f *rangeFacts, git gitRunner) (unreleasedState, error) {
	if !contains(f.changed, "CHANGELOG.md") {
		return unreleasedUntouched, nil
	}
	adds, err := addsUnreleased(f.rng, f.head, git)
	if err != nil || !adds {
		return unreleasedUntouched, err
	}
	release, err := isRelease(f.rng, f.changed, git)
	if err != nil {
		return unreleasedUntouched, err
	}
	if release {
		return unreleasedInRelease, nil
	}
	return unreleasedOutsideRelease, nil
}

// isRelease reports whether the range is a release, the only kind that may add lines inside
// `## [Unreleased]`: it deletes a changelog.d/ fragment, a file named <YYYY-MM-DD>-<slug>.md
// (scripts/gen-changelog.sh --write folds and deletes them), or it changes VERSION (the bump,
// CONTRIBUTING.md § 5). Either alone qualifies, so the fold and the bump pass whether they are pushed
// as one range or as two.
func isRelease(rng string, changed []string, git gitRunner) (bool, error) {
	if contains(changed, "VERSION") {
		return true, nil
	}
	deleted, err := git("diff", "--name-only", "--no-renames", "--diff-filter=D", rng, "--", FragmentDir+"/")
	if err != nil {
		return false, err
	}
	for _, p := range nonEmpty(deleted) {
		if strings.HasPrefix(p, FragmentDir+"/") && CheckName(strings.TrimPrefix(p, FragmentDir+"/")) == "" {
			return true, nil
		}
	}
	return false, nil
}
