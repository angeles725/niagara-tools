package checkbatsskips

import (
	"fmt"
	"regexp"
)

// ignoredLine is an allowlist line that holds no pattern: blank, or a `#` comment.
var ignoredLine = regexp.MustCompile(`^[ \t\n\v\f\r]*(#|$)`)

// Why an allowlist line is refused (stderr, after the tool prefix); the line follows as written.
const (
	notERE         = "allowlist line is not a valid ERE: %s"
	outsideDialect = "allowlist line uses %s, outside the dialect (POSIX ERE as Go regexp.CompilePOSIX reads it; tests/bats-skip-allowlist.txt header): %s"
)

// Allowlist is the patterns of an allowlist file, one POSIX extended regular expression per line.
type Allowlist []*regexp.Regexp

// ParseAllowlist compiles every pattern line of an allowlist text; fault is "" when every line is a
// pattern of the dialect, else why the first other line is refused.
//
// Declared difference from the bash tool, which asked GNU grep -E: the dialect is Go's POSIX ERE
// (regexp.CompilePOSIX), so a GNU-only extension Go does not compile (a back-reference, \w, a leading
// `*`) is not a valid line, and a construct Go compiles with another meaning (\<, \n, [[:word:]], a
// backslash in a bracket expression: dialect.go) is refused by name instead of matched.
// TestCommittedAllowlist pins the committed list to this dialect.
func ParseAllowlist(text string) (allow Allowlist, fault string) {
	// Mutation: SK10-last-line -- reading only newline-terminated lines drops a final line, so its skip reads UNLISTED
	for _, line := range lines(text) {
		if ignoredLine.MatchString(line) {
			continue
		}
		re, err := regexp.CompilePOSIX(line)
		if err != nil {
			return nil, fmt.Sprintf(notERE, line)
		}
		// Mutation: SK11-dialect -- dropping this check matches \< as a literal `<` and [[:word:]] as Go's class
		if construct := foreign(line); construct != "" {
			return nil, fmt.Sprintf(outsideDialect, construct, line)
		}
		allow = append(allow, re)
	}
	return allow, ""
}

// Allows reports whether one line matches the WHOLE reason (grep -x). Each line is its own pattern, so
// an alternation never spans two lines. The match is leftmost-longest: a pattern that can match the
// whole reason does so from offset 0 to its end.
func (a Allowlist) Allows(reason string) bool {
	for _, re := range a {
		// Mutation: SK3-unanchored -- accepting any match lets a pattern hide inside a longer reason
		if loc := re.FindStringIndex(reason); loc != nil && loc[0] == 0 && loc[1] == len(reason) {
			return true
		}
	}
	return false
}
