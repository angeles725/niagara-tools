package lint

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// WalkFollow is Walk for a rule that must not miss a file behind a symlink (lint-ntpl-manifest-deps: a
// skipped template would read as "nothing to check"). It selects, prunes and spells paths as Walk does, and
// differs below the root only:
//   - a symlink to a regular file is listed, under the link's own path, when sel selects the link's name;
//   - a symlink to a directory is entered, under the link's own path, unless sel prunes the link's name;
//   - a directory is entered once: one already walked under another spelling (a link back to an ancestor, a
//     second link to the same place) is not entered again, so a cycle ends and no file is listed twice;
//   - a symlink that leads nowhere, whose name sel selects, is a file nobody could read: it is returned as
//     the *WalkError (the first one wins, as in Walk). One with any other name is ignored.
//
// Walk itself is unchanged: every other rule keeps "symlinks below the root are neither followed nor listed".
func WalkFollow(root string, sel Selection) ([]string, error) {
	var files []string
	var first error
	fail := func(path string, err error) {
		if first == nil {
			first = &WalkError{Path: path, Err: err}
		}
	}
	entered := map[string]bool{}
	var walk func(dir string)
	walk = func(dir string) {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil {
			fail(dir, err)
			return
		}
		if entered[real] {
			return
		}
		entered[real] = true
		entries, err := os.ReadDir(dir)
		if err != nil {
			fail(dir, err)
			return
		}
		prefix := dir + "/"
		if strings.HasSuffix(dir, "/") {
			prefix = dir
		}
		for _, e := range entries {
			name, full := e.Name(), prefix+e.Name()
			mode := e.Type()
			if mode&os.ModeSymlink != 0 {
				st, err := os.Stat(full)
				if err != nil {
					if sel.selects(name) {
						// Mutation: WF-dangling -- ignoring a dangling link whose name is selected reads a template nobody could open as absent
						fail(full, err)
					}
					continue
				}
				mode = st.Mode().Type()
			}
			switch {
			case mode.IsDir():
				if !sel.prunes(name, full) {
					walk(full)
				}
			case mode.IsRegular() && sel.selects(name) && (len(sel.Paths) == 0 || matchAny(sel.Paths, full, true)):
				files = append(files, full)
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
