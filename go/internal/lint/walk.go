package lint

import (
	"fmt"
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

// WalkError is a directory the walk could not read; its text is the line GNU find printed for it
// under LC_ALL=C (exported by toolbelt/lib/awk-lint.sh), path quoted as findQuote spells it.
type WalkError struct {
	Path string
	Err  error
}

func (e *WalkError) Error() string { return "find: " + findQuote(e.Path) + ": " + strerror.Text(e.Err) }

// findQuote spells a path as GNU find quotes it in the C locale (quotearg locale style): always inside
// '...', with \' and \\, C escapes for \a \b \f \n \r \t \v, and 3-digit octal for every other byte
// outside printable ASCII (UTF-8 too: each byte is escaped on its own).
func findQuote(p string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for i := 0; i < len(p); i++ {
		c := p[i]
		switch esc := strings.IndexByte("\a\b\f\n\r\t\v", c); {
		case c == '\'' || c == '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case esc >= 0:
			b.WriteByte('\\')
			b.WriteByte("abfnrtv"[esc])
		case c < 0x20 || c >= 0x7f:
			fmt.Fprintf(&b, "\\%03o", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
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
				if !sel.prunes(name) {
					walk(prefix + name)
				}
			case t.IsRegular() && sel.selects(name):
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
