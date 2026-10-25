package citeparse

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	pageTok    = regexp.MustCompile(`^pp?\.?(\d+(?:-\d+)?)$`)
	moduleName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*(?:-[a-z]{2,3})?$`)
	zipEntry   = regexp.MustCompile("^`?(.+?\\.zip)!(\\S+?)`?(?:\\s+(.*))?$")
	moduleFile = regexp.MustCompile(`^(?:module(?:-include|-permissions)?\.xml|module\.palette|module\.lexicon|(?:[\w-]+\.)?gradle\.kts)$`)
)

// courseClause reads `course <path>[:ranges][ p<a>[-b]]…`: the path may hold spaces, so the words left
// after the page tokens are the target; the resolver finds the longest prefix of them that exists.
func courseClause(c, rest string) []Pointer {
	f := strings.Fields(rest)
	var pages []Range
	for len(f) > 1 {
		m := pageTok.FindStringSubmatch(f[len(f)-1])
		if m == nil {
			break
		}
		rs, _ := ParseRanges(m[1])
		pages, f = append(rs, pages...), f[:len(f)-1]
	}
	if len(f) == 0 {
		return unparsed(c, "no course file named")
	}
	path, lines := splitLines(strings.Join(f, " "))
	return []Pointer{{Kind: Course, Target: path, Lines: lines, Pages: pages, Raw: c}}
}

// archiveClause reads `<path>.zip!<entry>[:ranges]` (the path may hold spaces) as the zip and the entry in
// it, after `code` or `course`.
func archiveClause(c, rest string) []Pointer {
	m := zipEntry.FindStringSubmatch(rest)
	entry, lines := splitLines(strings.TrimPrefix(m[2], "/"))
	return []Pointer{{Kind: Archive, Target: m[1], File: entry, Lines: lines, Notes: strings.Fields(m[3]), Raw: c}}
}

// bitacoraClause reads `bitácora <slug>[ §section…]`: a client bitacora/<date>-<slug>.md file.
func bitacoraClause(c, rest string) []Pointer {
	f := strings.Fields(rest)
	if len(f) == 0 || !slugRe.MatchString(f[0]) {
		return unparsed(c, "bitácora slug not readable")
	}
	p := Pointer{Kind: Bitacora, Target: strings.TrimSuffix(f[0], ".md"), Raw: c}
	corpusLocator(&p, strings.Join(f[1:], " "))
	return []Pointer{p}
}

// kitClause reads `kit <clause>`: the clause, with a path made relative to the kit root.
func kitClause(c, rest string, aliases map[string]string) []Pointer {
	ps := clause(rest, aliases)
	for i := range ps {
		if ps[i].Kind == Path && !filepath.IsAbs(ps[i].Target) {
			ps[i].Target = "KIT/" + ps[i].Target
		}
		ps[i].Raw = c
	}
	return ps
}

// externalRetro reads `external retro <path to a retro>[ Δn…]` as the retro of that file name.
func externalRetro(c, rest string) []Pointer {
	path, more, _ := strings.Cut(rest, " ")
	return retroClause(c, strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), ".md")+" "+more))
}

// moduleRef reads `<module> <file>[ …]` as a code pointer when the first word names a module (a part
// suffix or a capital letter) or the file is a module build file, so prose (`see foo.java`) stays unparsed.
func moduleRef(c, word, rest string, aliases map[string]string) ([]Pointer, bool) {
	f := strings.Fields(rest)
	if len(f) == 0 || !moduleName.MatchString(word) {
		return nil, false
	}
	file, _ := splitLines(strings.Trim(f[0], "`"))
	if !looksPath(file) || !(strings.ContainsAny(word, "-ABCDEFGHIJKLMNOPQRSTUVWXYZ") || moduleFile.MatchString(file)) {
		return nil, false
	}
	return sourceClause(c, "code", word+" "+rest, aliases), true
}
