package vendorfloor

import (
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// vendoredSuffixes are the file names of a vendored browser library (find -name '*.js' -o '*.mjs' -o '*.cjs').
var vendoredSuffixes = []string{".js", ".mjs", ".cjs"}

// listVendored lists the vendored libraries under dir as
// `find <dir> -type f \( -name '*.js' -o -name '*.mjs' -o -name '*.cjs' \) | sort` does under LC_ALL=C:
// every sub-directory is entered (a dot directory too: a library hidden there still ships) and paths are
// dir + "/" + name. dir itself, being a symlink, is followed as the other lint roots are (find -H), with
// paths still spelled through the link; a symlink below it is neither followed nor listed, so a link that
// points back at an ancestor is never walked twice. Each directory that cannot be read is returned as
// the error find printed for it: the list is then incomplete and the caller fails closed.
// lint.Walk is not used: it prunes dot and *-worktrees sub-directories.
func listVendored(dir string) (files []string, errs []error) {
	var walk func(dir string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			errs = append(errs, &lint.WalkError{Path: dir, Err: err})
		}
		for _, e := range entries {
			child := dir + "/" + e.Name()
			switch t := e.Type(); {
			case t.IsDir():
				walk(child)
			case t.IsRegular() && vendored(e.Name()):
				files = append(files, child)
			}
		}
	}
	// Mutation: VF-symroot -- os.Lstat here lists nothing under a symlinked vendor/, which then reads "nothing to check"
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		walk(dir)
	}
	sort.Strings(files)
	return files, errs
}

func vendored(name string) bool {
	for _, suffix := range vendoredSuffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// dangling reports whether path is a symlink whose target does not exist.
func dangling(path string) bool {
	st, err := os.Lstat(path)
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		return false
	}
	_, err = os.Stat(path)
	return err != nil
}
