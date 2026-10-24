package citeparse

import (
	"regexp"
	"strings"
)

var (
	fileExt    = regexp.MustCompile(`\.[A-Za-z][A-Za-z0-9]{0,9}$`)
	identRe    = regexp.MustCompile(`^[A-Za-z][\w.-]*$`)
	className  = regexp.MustCompile(`^[A-Z][A-Za-z0-9_$]*$`)
	versionSeg = regexp.MustCompile(`^[\d.]+$`)
	sectionNum = regexp.MustCompile(`^\d+(?:\.\d+)*[a-z]?(?:-\d+)?`)
	aliasEq    = regexp.MustCompile("`([A-Z]{1,3})=([\\w./-]+)`")
	aliasDir   = regexp.MustCompile("`([A-Z]{1,3})/` = `([\\w./-]+)`")
	aliasCited = regexp.MustCompile("`([\\w./-]+)`, cited `([A-Z]{1,3}):`")
)

// aliases reads the per-card path aliases: `V=dir`, `G/` = `dir`, and (`file`, cited `CL:`).
func aliases(lines []string, fenced []bool) map[string]string {
	m := map[string]string{}
	for i, l := range lines {
		if fenced[i] {
			continue
		}
		for _, re := range []*regexp.Regexp{aliasEq, aliasDir} {
			for _, s := range re.FindAllStringSubmatch(l, -1) {
				m[s[1]] = strings.TrimSuffix(s[2], "/")
			}
		}
		for _, s := range aliasCited.FindAllStringSubmatch(l, -1) {
			m[s[2]] = s[1]
		}
	}
	return m
}

// expand rewrites `X/rest` to `<alias X>/rest` and a bare `X` to its alias.
func expand(path string, aliases map[string]string) string {
	head, rest, slash := strings.Cut(path, "/")
	if a, ok := aliases[head]; ok {
		if !slash {
			return a
		}
		return a + "/" + rest
	}
	return path
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

// sourceClause reads a clause that names a source file: code, devguide, guide, niagara-help, mem, or a
// path relative to a root (with lines, a § heading, or test names that must appear in it).
func sourceClause(c, word, rest string, aliases map[string]string) []Pointer {
	f := strings.Fields(rest)
	switch word {
	case "code":
		switch {
		case len(f) == 0:
			return unparsed(c, "no file named")
		case strings.Contains(f[0], "!"):
			return unparsed(c, "jar entry")
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

// pathPointer reads `<path>[:ranges][ § heading | token…]`.
func pathPointer(k Kind, text string, aliases map[string]string, raw string) Pointer {
	f := strings.Fields(text)
	path, lines := splitLines(strings.Trim(f[0], "`"))
	p := Pointer{Kind: k, Target: expand(path, aliases), Lines: lines, Raw: raw}
	extra := f[1:]
	if len(extra) > 0 && strings.HasPrefix(extra[0], "§") {
		h := strings.TrimSpace(strings.TrimPrefix(strings.Join(extra, " "), "§"))
		p.Sections = sectionRange(heading(h))
		return p
	}
	for _, w := range extra {
		if lines == nil && identRe.MatchString(w) {
			p.Tokens = append(p.Tokens, w)
		} else {
			p.Notes = append(p.Notes, w)
		}
	}
	return p
}

// heading reads a § reference: a section number (any title after it dropped) or a title, unquoted.
func heading(h string) string {
	if n := sectionNum.FindString(h); n != "" {
		return n
	}
	return strings.TrimSpace(strings.Trim(h, "\"`"))
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
