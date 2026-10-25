// Package changelog holds the CHANGELOG.md rules of niagara-tools: the changelog.d/ fragment contract,
// the `## [Unreleased]` section bounds and the changelog gate that .githooks/pre-push and CI run over a
// pushed range (scripts/changelog-gate.sh).
package changelog

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// FragmentDir holds one fragment per change, beside CHANGELOG.md at the repository root (as ledger.d/ sits
// beside the BUILD-STATE.src.md it feeds): two PRs that each add a fragment never touch the same line.
const FragmentDir = "changelog.d"

// UnreleasedHeader opens the section a release folds; the next "## [" line closes it.
const UnreleasedHeader = "## [Unreleased]"

var (
	fragmentName = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})-[a-z0-9][a-z0-9-]*\.md$`)
	kacHeading   = regexp.MustCompile(`^### (Added|Changed|Deprecated|Removed|Fixed|Security)( — \S.*)?$`)
	bullet       = regexp.MustCompile(`^- \S`)
)

// CheckName returns why base (a file name under changelog.d/) is not `<YYYY-MM-DD>-<slug>.md`, or "".
func CheckName(base string) string {
	m := fragmentName.FindStringSubmatch(base)
	if m == nil {
		return "the name is not <YYYY-MM-DD>-<slug>.md (slug: lower-case letters, digits and -)"
	}
	if _, err := time.Parse("2006-01-02", m[1]); err != nil {
		return fmt.Sprintf("the name's date %s is not a calendar date", m[1])
	}
	return ""
}

// CheckFragment returns why body is not a fragment, or "". A fragment starts on its first line with a
// Keep a Changelog heading (`### Added|Changed|Deprecated|Removed|Fixed|Security`, optionally ` — <title>`);
// every heading it holds has at least one `- ` bullet, and no other heading level appears, so folding it
// into `## [Unreleased]` can neither open a release section nor leave an empty subsection.
func CheckFragment(body string) string {
	if strings.TrimSpace(body) == "" {
		return "the fragment is empty"
	}
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	if !kacHeading.MatchString(lines[0]) {
		return fmt.Sprintf("line 1 is not a Keep a Changelog heading (### Added|Changed|Deprecated|Removed|Fixed|Security [— <title>]): %q", lines[0])
	}
	bullets := 0
	for i, l := range lines {
		if strings.HasPrefix(l, "#") {
			if !kacHeading.MatchString(l) {
				return fmt.Sprintf("line %d: only Keep a Changelog ### headings may appear: %q", i+1, l)
			}
			if i > 0 && bullets == 0 {
				return fmt.Sprintf("line %d: the heading before it has no '- ' bullet", i+1)
			}
			bullets = 0
			continue
		}
		if bullet.MatchString(l) {
			bullets++
		}
	}
	if bullets == 0 {
		return "the last heading has no '- ' bullet"
	}
	return ""
}

// UnreleasedBounds returns the 0-based index of the `## [Unreleased]` line and of the line that closes
// its section (the next "## [" line, or len(lines)); ok is false when the header is missing.
func UnreleasedBounds(lines []string) (start, end int, ok bool) {
	start = -1
	for i, l := range lines {
		t := strings.TrimRight(l, " \t\r")
		if start < 0 {
			if t == UnreleasedHeader {
				start = i
			}
			continue
		}
		if strings.HasPrefix(t, "## [") {
			return start, i, true
		}
	}
	return start, len(lines), start >= 0
}
