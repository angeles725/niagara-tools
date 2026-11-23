package citeparse

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const javaPath = "`?([\\w/.$…-]+\\.java)(?::(\\d[\\d,-]*))?`?"

// The prose citation forms, tried in this order; each match is blanked before the next form runs,
// so the generic backticked `path:lines` form never reads a span a specific form already took. A
// cross-reference file name may hold dots before `.md` (`BUILD-STATE.src.md` § X), read whole. A quoted
// phrase right after a backticked `path:lines` is its anchor.
var (
	vineModFirst = regexp.MustCompile(`vineflower\s+([A-Za-z][\w.-]*)\s+(\d+(?:\.\d+){2,3})\s+` + javaPath)
	vineVerFirst = regexp.MustCompile(`([A-Za-z][\w.]*-[a-z]+)\s+(\d+(?:\.\d+){2,3})\s+vineflower\s+` + javaPath)
	javapOf      = regexp.MustCompile("javap of `?([^`\\s]+\\.jar)`?(?:\\s+(\\d+(?:\\.\\d+){3}))?")
	devguideRef  = regexp.MustCompile("\\(devguide `([^`]+)`")
	certPath     = regexp.MustCompile("\\[CERT[^\\]]*\\]\\s+`?(organized/[^\\s`]+)")
	crossRef     = regexp.MustCompile("`?((?:[\\w.-]+/)*[\\w-]+(?:\\.[\\w-]+)*\\.md)`? (?:§§ (" + secListLoose + ")|§ (\"[^\"]+\"|`[^`]+`|" + secList + "|[^,;)\\]`|]+))")
	tickedPath   = regexp.MustCompile("`((?:[\\w.$… -]+/)+[\\w.$ -]*\\.\\w{1,10}):(\\d[\\d,:-]*)`(?:\\s+\"([^\"]+)\")?")
	tickedAlias  = regexp.MustCompile("`([A-Z]{1,3}):(\\d[\\d,-]*)`")
	// repoPath is a whole code span naming a repo-relative path under the roots retro
	// 2026-11-13-remove-dead-bash-lint-libs Δ1 lists; a span with a space, glob or ellipsis is not one path.
	repoPath = regexp.MustCompile("`((?:toolbelt|scripts|go|tests)/[\\w./-]*)`")
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
			if re == repoPath && !opensSpan(sc.lines[i], loc[0]) {
				continue // the backtick closes an earlier span: the text after it is prose, not a code span
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
		h, many := m[2]+m[3], m[2] != ""
		if !many && !strings.HasPrefix(h, "\"") && !strings.HasPrefix(h, "`") {
			h, _, _ = strings.Cut(h, ". ")
			h = strings.TrimRight(strings.TrimSpace(h), ".:")
		}
		s := sections(h, many)
		return Pointer{Kind: Path, Target: m[1], Sections: s}, len(s) > 0
	})
	take(tickedPath, func(m []string) (Pointer, bool) { // the path may hold spaces, so it is not split into words
		lines, _ := ParseRanges(strings.ReplaceAll(m[2], ":", "")) // `f:15,:82-88` repeats the colon
		return Pointer{Kind: Path, Target: expand(m[1], sc.aliases), Lines: lines, Anchor: strings.TrimSpace(m[3])}, onePath(m[1])
	})
	take(tickedAlias, func(m []string) (Pointer, bool) {
		a, ok := sc.aliases[m[1]]
		lines, _ := ParseRanges(m[2])
		return Pointer{Kind: Path, Target: a, Lines: lines}, ok
	})
	take(repoPath, func(m []string) (Pointer, bool) {
		return Pointer{Kind: RepoPath, Target: m[1]}, !strings.Contains(m[1], "..")
	})
	sort.SliceStable(hits, func(a, b int) bool { return hits[a].col < hits[b].col })
	out := make([]Pointer, len(hits))
	for k, h := range hits {
		out[k] = h.p
	}
	return out
}

// opensSpan reports whether the backtick at col of line opens a code span: an even number of backticks
// comes before it. Masking keeps columns, so col of the masked line is col of the line.
func opensSpan(line string, col int) bool {
	return strings.Count(line[:col], "`")%2 == 0
}

// onePath reports whether a backticked path is one path. A path with spaces is, when its first space falls
// after a `/` or it opens with a capitalised or numbered directory (`Slide Code/`, `Lab11 NDriver/`), and no
// ellipsis follows a space; else it is a label before a path (`code x-rt a/B.java`, `FMT …/y.sh`).
func onePath(p string) bool {
	sp := strings.IndexByte(p, ' ')
	switch {
	case sp < 0:
		return true
	case strings.Contains(p, " …"):
		return false
	}
	return strings.Contains(p[:sp], "/") || unicode.IsUpper(rune(p[0])) || unicode.IsDigit(rune(p[0]))
}
