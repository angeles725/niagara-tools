package lint

import (
	"os"
	"path"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// Selection is which files under a root a rule reads, as toolbelt/lib/scan-files.sh selects them.
type Selection struct {
	Names []string // find -name patterns, any of which selects a regular file
	Prune []string // sub-directory names never entered (dot and *-worktrees sub-directories never are)
	// Except are find -name patterns that drop a file Names selected (`-not -name '*.min.js'`).
	Except []string
	// Paths are find -path patterns on a selected file's whole path, one of which must match when set
	// (`-path '*/rc/*'`), and PrunePaths those of a sub-directory never entered (`--prune-path '*/rc/ext'`).
	// As in find, `*` and `?` match a "/" too; no other pattern character is supported.
	Paths, PrunePaths []string
}

func (s Selection) selects(name string) bool {
	return matchAny(s.Names, name, false) && !matchAny(s.Except, name, false)
}

// keeps reports whether s reads file p, found under root: its name and path select it and no directory
// between root and p is pruned.
func (s Selection) keeps(root, p string) bool {
	return s.selects(baseName(p)) && (len(s.Paths) == 0 || matchAny(s.Paths, p, true)) && !prunedFor(s, root, p)
}

// prunes also skips sibling "*-worktrees" checkouts, which would report every finding once per checkout.
// dir is the sub-directory's name and full its path as the walk spells it.
func (s Selection) prunes(dir, full string) bool {
	if strings.HasPrefix(dir, ".") || strings.HasSuffix(dir, "-worktrees") {
		return true
	}
	for _, p := range s.Prune {
		if p == dir {
			return true
		}
	}
	return matchAny(s.PrunePaths, full, true)
}

// matchAny reports whether one of patterns matches name: path.Match for a -name pattern, findPath for a
// -path pattern (whole set).
func matchAny(patterns []string, name string, whole bool) bool {
	for _, p := range patterns {
		if whole && findPath(p, name) {
			return true
		}
		if ok, _ := path.Match(p, name); ok && !whole {
			return true
		}
	}
	return false
}

// findPath is find -path matching (fnmatch without FNM_PATHNAME) for patterns of literal bytes, `*` (any
// run, "/" included) and `?` (any one byte).
func findPath(pattern, s string) bool {
	if pattern == "" {
		return s == ""
	}
	switch pattern[0] {
	case '*':
		for i := 0; i <= len(s); i++ {
			if findPath(pattern[1:], s[i:]) {
				return true
			}
		}
		return false
	case '?':
		return s != "" && findPath(pattern[1:], s[1:])
	}
	return s != "" && s[0] == pattern[0] && findPath(pattern[1:], s[1:])
}

// WalkError is a directory the walk could not read; its text is the line GNU find printed for it
// under LC_ALL=C (exported by toolbelt/lib/awk-lint.sh), path quoted as strerror.FindQuote spells it.
type WalkError struct {
	Path string
	Err  error
}

func (e *WalkError) Error() string {
	return "find: " + strerror.FindQuote(e.Path) + ": " + strerror.Text(e.Err)
}

// Walk lists the regular files under root that sel selects, sorted bytewise (LC_ALL=C sort), with
// paths spelled as find spells them (root + "/" + name, or root + name when root ends in "/").
// The root itself is never pruned and, being a symlink, is followed (find -H, issue #329) with paths
// still spelled from the root as given; symlinks below it are neither followed nor listed. A root that
// cannot be stat'ed (a dangling symlink) is a *WalkError, never an empty clean list. A directory it cannot read is skipped and the first one is returned
// as a *WalkError after the rest of the tree is walked: the caller must treat it as an incomplete scan
// (fail closed), never as clean.
func Walk(root string, sel Selection) ([]string, error) {
	var files []string
	var first error
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if first == nil {
				first = &WalkError{Path: dir, Err: err}
			}
			return
		}
		prefix := dir + "/"
		if strings.HasSuffix(dir, "/") {
			prefix = dir
		}
		for _, e := range entries {
			name := e.Name()
			switch t := e.Type(); {
			case t.IsDir():
				if !sel.prunes(name, prefix+name) {
					walk(prefix + name)
				}
			case t.IsRegular() && sel.selects(name) && (len(sel.Paths) == 0 || matchAny(sel.Paths, prefix+name, true)):
				files = append(files, prefix+name)
			}
		}
	}
	if st, err := os.Stat(root); err != nil {
		return nil, &WalkError{Path: root, Err: err}
	} else if st.IsDir() {
		walk(root)
	}
	sort.Strings(files)
	return files, first
}
