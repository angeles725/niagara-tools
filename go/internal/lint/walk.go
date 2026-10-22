package lint

import (
	"errors"
	"os"
	"path"
	"sort"
	"strings"
	"syscall"
)

// Selection is which files under a root a rule reads, as toolbelt/lib/scan-files.sh selects them.
type Selection struct {
	Names []string // find -name patterns, any of which selects a regular file
	Prune []string // sub-directory names never entered (dot and *-worktrees sub-directories never are)
}

func (s Selection) selects(name string) bool {
	for _, p := range s.Names {
		if ok, _ := path.Match(p, name); ok {
			return ok
		}
	}
	return false
}

// prunes also skips sibling "*-worktrees" checkouts, which would report every finding once per checkout.
func (s Selection) prunes(dir string) bool {
	if strings.HasPrefix(dir, ".") || strings.HasSuffix(dir, "-worktrees") {
		return true
	}
	for _, p := range s.Prune {
		if p == dir {
			return true
		}
	}
	return false
}

// WalkError is a directory the walk could not read; its text is the line find printed for it.
type WalkError struct {
	Path string
	Err  error
}

func (e *WalkError) Error() string { return "find: '" + e.Path + "': " + strerror(e.Err) }

// strerror is the C library text of err's errno (Go's text with a capital first letter), as find
// and awk print it in the C locale.
func strerror(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	s := errno.Error()
	return strings.ToUpper(s[:1]) + s[1:]
}

// Walk lists the regular files under root that sel selects, sorted bytewise (LC_ALL=C sort), with
// paths spelled as find spells them (root + "/" + name, or root + name when root ends in "/").
// The root itself is never pruned and, being a symlink, is not followed (find -P); symlinks below it
// are neither followed nor listed. A directory it cannot read is skipped and the first one is returned
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
				if !sel.prunes(name) {
					walk(prefix + name)
				}
			case t.IsRegular() && sel.selects(name):
				files = append(files, prefix+name)
			}
		}
	}
	if st, err := os.Lstat(root); err != nil {
		return nil, &WalkError{Path: root, Err: err}
	} else if st.IsDir() {
		walk(root)
	}
	sort.Strings(files)
	return files, first
}
