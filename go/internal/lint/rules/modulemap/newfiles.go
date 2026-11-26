package modulemap

import (
	"path"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// baseOption is the option that names the ref MM8 compares the module against.
const baseOption = "--base"

// Configure is the rule with the --base value of its run ("" when not given: MM8 does not run).
func (rule) Configure(a lint.Args) (lint.Configured, error) {
	return rule{base: a.Option}, nil
}

// mm8 reports every file the candidate added since the base ref that a row of the map does not list, in path
// order: WARN in a legacy module, FAIL in a strict one. It covers the files the map covers beyond MM4's: srcTest
// Java files, SPA tests under webtest/ and vendored front-end files (src/rc vendor/, ext/, *.min.*); a new source
// file in no row is MM4's row alone. The base ref comes from the caller (--base); without one MM8 does not run,
// and a ref that does not resolve in the module's repository is one INFO row.
func (s *scan) mm8(rows []row) {
	if s.base == "" {
		return
	}
	added, ok := addedSince(s.root, s.base)
	if !ok {
		s.add(lint.Info, s.root, 0, "MM8: new-file check skipped: base ref "+s.base+" does not resolve here "+
			"(needs a base ref in the module's repository, e.g. --base origin/main)")
		return
	}
	var fresh []string
	for _, f := range s.files {
		if added[f] && coveredBeyondMM4(f) && !listedBy(rows, f) {
			fresh = append(fresh, f)
		}
	}
	sort.Strings(fresh)
	for _, f := range fresh {
		s.add(s.strictSev(), s.abs(f), 0, "MM8: new file in no concern (list it in the Files or Tests of the concern that owns it, in the same change)")
	}
}

// addedSince is the set of files under root (relative to it) that exist now and not at ref: added to the index or
// committed since, or untracked and not ignored. ok is false when ref does not name a commit of root's repository.
func addedSince(root, ref string) (set map[string]bool, ok bool) {
	if _, ok := gitcmd.Run(root, nil, "rev-parse", "--verify", "--quiet", ref+"^{commit}"); !ok {
		return nil, false
	}
	diff, ok1 := gitcmd.Run(root, nil, "diff", "--name-only", "--no-renames", "--diff-filter=A", "--relative", "-z", ref, "--")
	other, ok2 := gitcmd.Run(root, nil, "ls-files", "-z", "--others", "--exclude-standard")
	if !ok1 || !ok2 {
		return nil, false
	}
	set = map[string]bool{}
	for _, list := range []string{diff, other} {
		for _, p := range strings.Split(list, "\x00") {
			if p != "" {
				set[p] = true
			}
		}
	}
	return set, true
}

// coveredBeyondMM4: a part file the map lists but MM4 does not require: a srcTest Java file, an SPA test (a
// *.test.mjs or *.test.js under webtest/) or a vendored src/rc file.
func coveredBeyondMM4(rel string) bool {
	switch {
	case isSource(rel):
		return false
	case inSrcTest(rel):
		return strings.HasSuffix(rel, ".java")
	case strings.Contains("/"+rel, "/webtest/"):
		return strings.HasSuffix(rel, ".test.mjs") || strings.HasSuffix(rel, ".test.js")
	}
	seg := strings.Split(rel, "/")
	if len(seg) < 4 || seg[1] != "src" || seg[2] != "rc" {
		return false
	}
	if strings.Contains(seg[len(seg)-1], ".min.") {
		return true
	}
	for _, d := range seg[3 : len(seg)-1] {
		if d == "vendor" || d == "ext" {
			return true
		}
	}
	return false
}

// listedBy: an entry of some row's Files or Tests names rel exactly, or is a glob matching it (never a .java file).
func listedBy(rows []row, rel string) bool {
	for _, r := range rows {
		for _, e := range append(append([]string{}, r.files...), r.tests...) {
			clean := path.Clean(e)
			if clean == rel || (strings.ContainsAny(e, "*?[") && !strings.HasSuffix(rel, ".java") && matchGlob(clean, rel)) {
				return true
			}
		}
	}
	return false
}
