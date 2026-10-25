package citeparse

import (
	"regexp"
	"strings"
)

// secNum is a section number or a range of two (`11.1-11.3`, `2.4a`, `12.3-§12.5`); secList and secListLoose
// are lists of them after one § or after §§.
const (
	secNum       = `\d+(?:\.\d+)*(?:\.?[a-z])?(?:[–-]§?\d+(?:\.\d+)*)?`
	secList      = secNum + `(?:\s*,\s*§\s*` + secNum + `)*`
	secListLoose = secNum + `(?:\s*,\s*§?\s*` + secNum + `)*`
)

var (
	fileExt    = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]{0,9}$`)
	identRe    = regexp.MustCompile(`^[A-Za-z][\w.-]*$`)
	className  = regexp.MustCompile(`^[A-Z][A-Za-z0-9_$]*$`)
	versionSeg = regexp.MustCompile(`^[\d.]+$`)
	jarVersion = regexp.MustCompile(`^\d+(?:\.\d+){3}$`)
	sectionNum = regexp.MustCompile(secNum)
	aliasEq    = regexp.MustCompile("`([A-Z]{1,3})=([\\w./-]+)`")
	aliasDir   = regexp.MustCompile("`([A-Z]{1,3})/` = `([\\w./-]+)`")
	aliasCited = regexp.MustCompile("`([\\w./-]+)`, cited `([A-Z]{1,3}):`")
	aliasHelp  = regexp.MustCompile("\\b([A-Z]{1,3})\\s+=\\s+niagara-help\\s+(\\d+(?:\\.\\d+)+)\\s+`([\\w./-]+)`")
	plainWord  = regexp.MustCompile(`^[A-Za-z][a-z]*$`)
	copyWords  = map[string]bool{"vineflower": true, "decompiled": true, "extracted": true}

	// sectionList is a list of section numbers, each after the first opening with §; sectionListLoose
	// (after §§) also takes a bare number after a comma.
	sectionList      = regexp.MustCompile(`^` + secList)
	sectionListLoose = regexp.MustCompile(`^` + secListLoose)
)

// maxAliasDepth bounds the expansion of an alias defined by another (`V=R/vineflower`).
const maxAliasDepth = 3

// aliases reads the per-card path aliases: `V=dir`, `G/` = `dir`, (`file`, cited `CL:`), and a legend
// GD = niagara-help <ver> `dir/`, that may wrap a line.
func aliases(lines []string, fenced []bool) map[string]string {
	m := map[string]string{}
	var prose []string
	for i, l := range lines {
		if fenced[i] {
			continue
		}
		prose = append(prose, l)
		for _, re := range []*regexp.Regexp{aliasEq, aliasDir} {
			for _, s := range re.FindAllStringSubmatch(l, -1) {
				m[s[1]] = strings.TrimSuffix(s[2], "/")
			}
		}
		for _, s := range aliasCited.FindAllStringSubmatch(l, -1) {
			m[s[2]] = s[1]
		}
	}
	for _, s := range aliasHelp.FindAllStringSubmatch(strings.Join(prose, " "), -1) {
		m[s[1]] = "niagara-help/versions/" + s[2] + "/" + strings.TrimSuffix(s[3], "/")
	}
	return m
}

// expand rewrites `X/rest` to `<alias X>/rest` and a bare `X` to its alias, through aliases of aliases,
// and an ASCII `...` segment to the `…` ellipsis.
func expand(path string, aliases map[string]string) string {
	for i := 0; i < maxAliasDepth; i++ {
		head, rest, slash := strings.Cut(path, "/")
		a, ok := aliases[head]
		if !ok {
			break
		}
		if path = a; slash {
			path = a + "/" + rest
		}
	}
	if rest, ok := strings.CutPrefix(path, ".../"); ok {
		path = "…/" + rest
	}
	return strings.ReplaceAll(path, "/.../", "/…/")
}

// splitLines cuts `path:ranges` into the path and its ranges (nil when the suffix is no range list).
func splitLines(s string) (string, []Range) {
	if i := strings.LastIndexByte(s, ':'); i > 0 {
		if rs, ok := ParseRanges(s[i+1:]); ok {
			return s[:i], rs
		}
	}
	return s, nil
}

func looksPath(w string) bool {
	p, _ := splitLines(w)
	return strings.Contains(p, "/") || fileExt.MatchString(p)
}

// sourceClause reads a clause that names a source file: code, javap, devguide, guide, niagara-help, mem,
// or a path relative to a root (with lines, a § heading, or test names that must appear in it).
func sourceClause(c, word, rest string, aliases map[string]string) []Pointer {
	f := strings.Fields(rest)
	switch word {
	case "code":
		for i := range f {
			f[i] = strings.Trim(f[i], "`,;")
		}
		if len(f) >= 3 && copyWords[f[1]] && looksPath(f[2]) {
			p := pathPointer(Vineflower, strings.Join(f[2:], " "), aliases, c)
			p.Target, p.File = f[0], p.Target
			if f[1] != "vineflower" {
				p.Kind, p.File = Code, f[1]+"/"+p.File
			}
			return []Pointer{p}
		}
		switch {
		case len(f) == 0:
			return unparsed(c, "no file named")
		case f[0] == "javap":
			return javapClause(c, f[1:])
		case strings.Contains(f[0], ".jar!"):
			return []Pointer{jarEntry(c, f[0], f[1:])}
		case strings.Contains(f[0], "!"):
			return unparsed(c, "archive entry")
		case len(f) >= 2 && !strings.ContainsAny(f[0], "/.:") && className.MatchString(f[1]):
			return []Pointer{{Kind: Code, Target: f[0], File: f[1] + ".java", Notes: f[2:], Raw: c}}
		case len(f) >= 2 && !strings.ContainsAny(f[0], "/.:") && looksPath(f[1]):
			p := pathPointer(Code, strings.Join(f[1:], " "), aliases, c)
			p.Target, p.File = f[0], p.Target
			return []Pointer{p}
		}
		p := pathPointer(Code, rest, aliases, c)
		p.Target, p.File = "", p.Target
		return []Pointer{p}
	case "vineflower":
		if len(f) == 0 || !looksPath(strings.Trim(f[0], "`")) {
			return unparsed(c, "no file named")
		}
		p := pathPointer(Vineflower, rest, aliases, c)
		p.Target, p.File = "", p.Target
		return []Pointer{p}
	case "devguide", "guide":
		if len(f) == 0 {
			return unparsed(c, "no file named")
		}
		return []Pointer{pathPointer(Kind(word), rest, aliases, c)}
	case "niagara-help":
		if len(f) < 2 {
			return unparsed(c, "niagara-help needs <version> <path>")
		}
		p := pathPointer(Help, strings.Join(f[1:], " "), aliases, c)
		p.Version = f[0]
		return []Pointer{p}
	case "javap":
		return javapClause(c, f)
	case "mem", "memory":
		if len(f) == 0 {
			return unparsed(c, "no memory named")
		}
		return []Pointer{{Kind: Mem, Target: strings.TrimSuffix(f[0], ".md"), Notes: f[1:], Raw: c}}
	}
	if looksPath(word) {
		if why := notAPath(word); why != "" {
			return unparsed(c, why)
		}
		return []Pointer{pathPointer(Path, c, aliases, c)}
	}
	return unparsed(c, "kind not covered")
}

// javapClause reads `javap <x.jar>[!entry][ <version>]`, the tag form of the prose "javap of `x.jar`": the
// jar is checked, what javap printed is not. A class name with no jar names nothing to check.
func javapClause(c string, f []string) []Pointer {
	if len(f) == 0 {
		return unparsed(c, "no jar named")
	}
	jar := strings.Trim(f[0], "`,;")
	switch {
	case strings.Contains(jar, ".jar!"):
		return []Pointer{jarEntry(c, jar, f[1:])}
	case !strings.HasSuffix(jar, ".jar"):
		return unparsed(c, "no jar named")
	}
	p := Pointer{Kind: Javap, Target: jar, Notes: f[1:], Raw: c}
	if len(f) > 1 && jarVersion.MatchString(f[1]) {
		p.Version, p.Notes = f[1], f[2:]
	}
	return []Pointer{p}
}

// jarEntry reads `x.jar!entry[:ranges]` as the jar and the entry in it.
func jarEntry(c, word string, notes []string) Pointer {
	jar, entry, _ := strings.Cut(word, ".jar!")
	entry, lines := splitLines(strings.TrimPrefix(entry, "/"))
	return Pointer{Kind: Jar, Target: jar + ".jar", File: entry, Lines: lines, Notes: notes, Raw: c}
}

// pathPointer reads `<path>[:ranges][ § heading | token…]`.
func pathPointer(k Kind, text string, aliases map[string]string, raw string) Pointer {
	f := strings.Fields(text)
	path, lines := splitLines(strings.Trim(f[0], "`,;"))
	p := Pointer{Kind: k, Target: expand(path, aliases), Lines: lines, Raw: raw}
	extra := f[1:]
	if len(extra) > 0 && strings.HasPrefix(extra[0], "§") {
		h := strings.TrimPrefix(strings.Join(extra, " "), "§")
		p.Sections = sections(strings.TrimLeft(h, "§ "), strings.HasPrefix(h, "§"))
		return p
	}
	for len(extra) > 0 && strings.HasPrefix(extra[0], ":") {
		rs, ok := ParseRanges(strings.Trim(extra[0], ":`,;"))
		if !ok {
			break
		}
		p.Lines, extra = append(p.Lines, rs...), extra[1:]
	}
	if len(extra) >= 2 && p.Lines == nil && (extra[0] == "line" || extra[0] == "lines") {
		if rs, ok := ParseRanges(extra[1]); ok {
			p.Lines, extra = rs, extra[2:]
		}
	}
	if p.Lines != nil {
		p.Anchor, extra = anchor(extra)
	}
	prose := p.Lines != nil
	for _, w := range extra {
		// A plain word (`list`, `Model`) opens prose: it and every word after it are notes.
		prose = prose || plainWord.MatchString(w)
		if prose || !identRe.MatchString(w) {
			p.Notes = append(p.Notes, w)
		} else {
			p.Tokens = append(p.Tokens, w)
		}
	}
	return p
}

// anchor reads a quoted phrase that opens words (`"Absolute Logoff" read` → Absolute Logoff) and returns
// the words after it; words that open no closed quote are returned as they are.
func anchor(words []string) (string, []string) {
	text := strings.Join(words, " ")
	rest, ok := strings.CutPrefix(text, `"`)
	if !ok {
		return "", words
	}
	phrase, after, closed := strings.Cut(rest, `"`)
	if !closed || strings.TrimSpace(phrase) == "" {
		return "", words
	}
	return strings.TrimSpace(phrase), strings.Fields(after)
}

// sections reads a § reference: a list of section numbers (any title after them dropped; a bare number
// after a comma only under §§, many), each range as its two ends, or one title, unquoted.
func sections(h string, many bool) []string {
	list := sectionList
	if many {
		list = sectionListLoose
	}
	l := list.FindString(h)
	if l == "" {
		if t := strings.TrimSpace(strings.Trim(h, "\"`")); t != "" {
			return []string{t}
		}
		return nil
	}
	var out []string
	for _, n := range sectionNum.FindAllString(l, -1) {
		out = append(out, sectionRange(n)...)
	}
	return out
}

// notAPath says why a path-like word is not one file: a glob, or a list of version numbers.
func notAPath(w string) string {
	if strings.ContainsAny(w, "*?") {
		return "glob, not one file"
	}
	for _, seg := range strings.Split(w, "/") {
		if !versionSeg.MatchString(seg) {
			return ""
		}
	}
	return "version list, not a path"
}
