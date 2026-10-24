package citeparse

import (
	"regexp"
	"sort"
	"strings"
)

const javaPath = "`?([\\w/.$…-]+\\.java)(?::(\\d[\\d,-]*))?`?"

// The prose citation forms, tried in this order; each match is blanked before the next form runs,
// so the generic backticked `path:lines` form never reads a span a specific form already took.
var (
	vineModFirst = regexp.MustCompile(`vineflower\s+([A-Za-z][\w.-]*)\s+(\d+(?:\.\d+){2,3})\s+` + javaPath)
	vineVerFirst = regexp.MustCompile(`([A-Za-z][\w.]*-[a-z]+)\s+(\d+(?:\.\d+){2,3})\s+vineflower\s+` + javaPath)
	javapOf      = regexp.MustCompile("javap of `?([^`\\s]+\\.jar)`?(?:\\s+(\\d+(?:\\.\\d+){3}))?")
	devguideRef  = regexp.MustCompile("\\(devguide `([^`]+)`")
	certPath     = regexp.MustCompile("\\[CERT[^\\]]*\\]\\s+`?(organized/[^\\s`]+)")
	crossRef     = regexp.MustCompile("`?((?:[\\w.-]+/)*[\\w-]+\\.md)`? § (\"[^\"]+\"|`[^`]+`|[^,;)\\]`|]+)")
	tickedPath   = regexp.MustCompile("`((?:[\\w.$…-]+/)+[\\w.$-]*\\.\\w{1,10}):(\\d[\\d,-]*)`")
	tickedAlias  = regexp.MustCompile("`([A-Z]{1,3}):(\\d[\\d,-]*)`")
)

type hit struct {
	col int
	p   Pointer
}

// inline reads the prose citations of line i from its masked copy (tags already blanked).
func (sc *scan) inline(i int) []Pointer {
	var hits []hit
	take := func(re *regexp.Regexp, mk func(m []string) (Pointer, bool)) {
		l := sc.masked[i]
		for _, loc := range re.FindAllStringSubmatchIndex(l, -1) {
			m := make([]string, len(loc)/2)
			for k := range m {
				if loc[2*k] >= 0 {
					m[k] = l[loc[2*k]:loc[2*k+1]]
				}
			}
			p, ok := mk(m)
			if !ok {
				continue
			}
			p.Line, p.Raw = i+1, strings.TrimSpace(m[0])
			hits = append(hits, hit{loc[0], p})
			sc.masked[i] = sc.masked[i][:loc[0]] + strings.Repeat(" ", loc[1]-loc[0]) + sc.masked[i][loc[1]:]
		}
	}
	vine := func(m []string) (Pointer, bool) {
		lines, _ := ParseRanges(m[4])
		return Pointer{Kind: Vineflower, Target: m[1], Version: m[2], File: m[3], Lines: lines}, true
	}
	take(vineModFirst, vine)
	take(vineVerFirst, vine)
	take(javapOf, func(m []string) (Pointer, bool) {
		return Pointer{Kind: Javap, Target: m[1], Version: m[2]}, true
	})
	take(devguideRef, func(m []string) (Pointer, bool) {
		return pathPointer(Devguide, m[1], sc.aliases, ""), true
	})
	take(certPath, func(m []string) (Pointer, bool) {
		return pathPointer(Path, strings.TrimRight(m[1], ".,;)"), sc.aliases, ""), true
	})
	take(crossRef, func(m []string) (Pointer, bool) {
		h := m[2]
		if !strings.HasPrefix(h, "\"") && !strings.HasPrefix(h, "`") {
			h, _, _ = strings.Cut(h, ". ")
			h = strings.TrimRight(strings.TrimSpace(h), ".:")
		}
		return Pointer{Kind: Path, Target: m[1], Sections: []string{heading(h)}}, heading(h) != ""
	})
	take(tickedPath, func(m []string) (Pointer, bool) {
		return pathPointer(Path, m[1]+":"+m[2], sc.aliases, ""), true
	})
	take(tickedAlias, func(m []string) (Pointer, bool) {
		a, ok := sc.aliases[m[1]]
		lines, _ := ParseRanges(m[2])
		return Pointer{Kind: Path, Target: a, Lines: lines}, ok
	})
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].col < hits[b].col })
	out := make([]Pointer, len(hits))
	for k, h := range hits {
		out[k] = h.p
	}
	return out
}
