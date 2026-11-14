package checkbatsskips

import "regexp"

// ignoredLine is an allowlist line that holds no pattern: blank, or a `#` comment.
var ignoredLine = regexp.MustCompile(`^[ \t\n\v\f\r]*(#|$)`)

// Allowlist is the patterns of an allowlist file, one POSIX extended regular expression per line.
type Allowlist []*regexp.Regexp

// ParseAllowlist compiles every pattern line of an allowlist text; ok is false at the first line that
// is not a valid ERE, returned as bad.
//
// Declared difference from the bash tool, which asked GNU grep -E: the dialect is Go's POSIX ERE
// (regexp.CompilePOSIX), so a GNU-only extension (a back-reference, \w, \<, a leading `*`) is rejected
// as an invalid line instead of matched. TestCommittedAllowlist pins the committed list to this dialect.
func ParseAllowlist(text string) (allow Allowlist, bad string, ok bool) {
	// Mutation: SK10-last-line -- reading only newline-terminated lines drops a final line, so its skip reads UNLISTED
	for _, line := range lines(text) {
		if ignoredLine.MatchString(line) {
			continue
		}
		re, err := regexp.CompilePOSIX(line)
		if err != nil {
			return nil, line, false
		}
		allow = append(allow, re)
	}
	return allow, "", true
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
