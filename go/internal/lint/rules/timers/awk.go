package timers

import (
	"regexp"
	"strings"
)

// The checks are ports of gawk programs, so their matching keeps gawk's semantics: POSIX
// leftmost-longest matches, "." matching a newline, and "while (match(t, re)) t = substr(t, ...)"
// loops in which ^ anchors at the start of the shrinking remainder.

// rxCache compiles each pattern once per run (the per-field and per-method patterns are built at run time).
type rxCache map[string]*regexp.Regexp

func (c rxCache) get(pattern string) *regexp.Regexp {
	re, ok := c[pattern]
	if !ok {
		re = regexp.MustCompile("(?s)" + pattern)
		re.Longest()
		c[pattern] = re
	}
	return re
}

func (c rxCache) has(s, pattern string) bool { return c.get(pattern).MatchString(s) }

// find is awk's match(): the leftmost-longest match of pattern in s, or ok == false.
func (c rxCache) find(s, pattern string) (m string, ok bool) {
	loc := c.get(pattern).FindStringIndex(s)
	if loc == nil {
		return "", false
	}
	return s[loc[0]:loc[1]], true
}

// sub is awk's sub(pattern, "", s): the first match removed.
func (c rxCache) sub(s, pattern string) string {
	loc := c.get(pattern).FindStringIndex(s)
	if loc == nil {
		return s
	}
	return s[:loc[0]] + s[loc[1]:]
}

// gsub is awk's gsub(pattern, "", s): every match removed.
func (c rxCache) gsub(s, pattern string) string { return c.get(pattern).ReplaceAllLiteralString(s, "") }

// each runs the awk loop `while (match(t, re)) { pre = text before the match; m = the match;
// t = the rest }`, calling fn(pre, m, rest) until fn returns false or nothing matches.
func (c rxCache) each(t, pattern string, fn func(pre, m, rest string) bool) {
	re := c.get(pattern)
	for {
		loc := re.FindStringIndex(t)
		if loc == nil {
			return
		}
		pre, m := t[:loc[0]], t[loc[0]:loc[1]]
		t = t[loc[1]:]
		if !fn(pre, m, t) {
			return
		}
	}
}

const (
	sp      = `[[:space:]]`
	ident   = `[A-Za-z_][A-Za-z0-9_]*`
	notWord = `(^|[^A-Za-z0-9_])`  // the start of t, or a character that cannot end an identifier
	notRef  = `(^|[^A-Za-z0-9_.])` // the same, also excluding a qualified a.name
)

// calls lists the same-file calls of body as the lifecycle awk walks find them: every
// identifier followed by "(" that is not a qualified .name call, in order.
func (c rxCache) calls(body string, fn func(name string)) {
	c.each(body, ident+sp+`*\(`, func(pre, m, _ string) bool {
		if !strings.HasSuffix(pre, ".") {
			fn(strings.TrimRight(m[:len(m)-1], " \t\n\v\f\r"))
		}
		return true
	})
}
