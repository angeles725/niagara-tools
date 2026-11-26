package uxversion

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// A sub-tree version is declared by an HTML comment in any scanned file: `<!-- ux-version: <dir>/ = <ver> -->`.
// It tells the lint that a separately built app under <dir>/ is versioned apart from the module: a reference whose
// path (as written) starts with <dir>/ or holds /<dir>/ must carry ?v=<ver>, not defaultModuleVersion.
var declRe = regexp.MustCompile(`<!--\s*ux-version:\s*([^\s=<>]+/)\s*=\s*([^\s<>]+?)\s*-->`)

// subtree is one declared sub-tree: its directory (with a trailing slash), its version and where it is declared.
type subtree struct {
	dir, ver, file string
	line           int
}

// subtrees are the declared sub-trees of a module, longest directory first.
type subtrees []subtree

// source is one scanned file.
type source struct {
	path  string
	lines []string
}

// declarations reads the ux-version comments of every file. Two declarations of one directory that differ are
// a subtree-version FAIL at the later one (the earlier stays in force, so one conflict is reported once).
func declarations(srcs []source) (subtrees, []lint.Finding) {
	var subs subtrees
	var out []lint.Finding
	for _, s := range srcs {
		for i, l := range s.lines {
			for _, m := range declRe.FindAllStringSubmatch(l, -1) {
				dir := strings.TrimLeft(strings.TrimPrefix(m[1], "./"), "/")
				if prev, ok := subs.byDir(dir); !ok {
					subs = append(subs, subtree{dir, m[2], s.path, i + 1})
				} else if prev.ver != m[2] {
					out = append(out, lint.Finding{Path: s.path, Line: i + 1, Severity: lint.Fail, Msg: fmt.Sprintf(
						"%s: %s is declared ?v=%s here but ?v=%s in %s:%d", checkSubtree, dir, m[2], prev.ver, prev.file, prev.line)})
				}
			}
		}
	}
	sort.SliceStable(subs, func(i, j int) bool { return len(subs[i].dir) > len(subs[j].dir) })
	return subs, out
}

func (s subtrees) byDir(dir string) (subtree, bool) {
	for _, t := range s {
		if t.dir == dir {
			return t, true
		}
	}
	return subtree{}, false
}

// find is the sub-tree a reference lies under (the longest declared directory wins).
func (s subtrees) find(ref string) (subtree, bool) {
	p, _, _ := strings.Cut(ref, "?")
	p, _, _ = strings.Cut(p, "#")
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimLeft(p, "/")
	for _, t := range s {
		if strings.HasPrefix(p, t.dir) || strings.Contains(p, "/"+t.dir) {
			return t, true
		}
	}
	return subtree{}, false
}
