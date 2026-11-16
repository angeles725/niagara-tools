package rcscan

import (
	"regexp"
	"strings"
)

// pat is an awk ERE compiled with gawk's leftmost-longest match, so a match's span (RSTART, RLENGTH) is gawk's,
// plus literals one of which every match contains: a line without it never reaches the regexp engine, whose linear NFA is
// still slow on the 100-KB data-URI and minified lines of a real page.
type pat struct {
	lits  []string       // one of them is in every match; none: no pre-check
	r, lr *regexp.Regexp // leftmost-first for a yes/no match (it stops at the first), leftmost-longest for a span
}

func re(expr string, lits ...string) *pat {
	lr := regexp.MustCompile(expr)
	lr.Longest()
	return &pat{lits, regexp.MustCompile(expr), lr}
}

func (p *pat) on(s string) bool {
	for _, l := range p.lits {
		if strings.Contains(s, l) {
			return true
		}
	}
	return len(p.lits) == 0
}

func (p *pat) MatchString(s string) bool { return p.on(s) && p.r.MatchString(s) }

// The span methods below use lr.

func (p *pat) FindString(s string) string {
	if !p.on(s) {
		return ""
	}
	return p.lr.FindString(s)
}

func (p *pat) FindStringIndex(s string) []int {
	if !p.on(s) {
		return nil
	}
	return p.lr.FindStringIndex(s)
}

func (p *pat) FindAllString(s string) []string {
	if !p.on(s) {
		return nil
	}
	return p.lr.FindAllString(s, -1)
}

func (p *pat) ReplaceAllString(s, with string) string {
	if !p.on(s) {
		return s
	}
	return p.lr.ReplaceAllString(s, with)
}
