package lint

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNotRegular is the WalkError reason of an entry WalkFollow selects by name that is neither a regular
// file nor a directory (a FIFO, a socket, a device, or a link to one): nobody can read it as a file.
var ErrNotRegular = errors.New("not a regular file")

// WalkFollow is Walk for a rule that must not miss a file behind a symlink (lint-ntpl-manifest-deps: a
// skipped template would read as "nothing to check"). It selects, prunes and spells paths as Walk does, and
// differs below the root only:
//   - a symlink is followed only when its real target is inside the root's real path; every other link
//     that is a directory, or whose name sel selects, is not followed and is returned in outside (sorted),
//     so the caller can say so: a link to an ancestor or to "/" would read a sibling module or the whole
//     file system;
//   - a symlink to a regular file is listed, under the link's own path, when sel selects the link's name;
//   - a symlink to a directory is entered, under the link's own path, unless sel prunes the link's name or
//     its target (the target's name, and its path spelled from the root), so out -> build stays pruned;
//   - a directory is entered once and a file listed once: one already reached under another spelling (a
//     link back to an ancestor, a second link to the same place) is not entered or listed again, so a
//     cycle ends;
//   - a symlink that leads nowhere, or an entry that is neither a regular file nor a directory
//     (ErrNotRegular), whose name sel selects, is a file nobody could read: it is returned as the
//     *WalkError (the first one wins, as in Walk). One with any other name is ignored.
//
// Walk itself is unchanged: every other rule keeps "symlinks below the root are neither followed nor listed".
func WalkFollow(root string, sel Selection) (files, outside []string, err error) {
	if st, err := os.Stat(root); err != nil {
		return nil, nil, &WalkError{Path: root, Err: err}
	} else if !st.IsDir() {
		return nil, nil, nil
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, nil, &WalkError{Path: root, Err: err}
	}
	w := follower{root: root, realRoot: realRoot, sel: sel, seen: map[string]bool{}}
	w.walk(root)
	sort.Strings(w.files)
	sort.Strings(w.outside)
	return w.files, w.outside, w.first
}

// follower is one WalkFollow run: seen holds the real path of every directory entered and file listed.
type follower struct {
	root, realRoot string
	sel            Selection
	seen           map[string]bool
	files, outside []string
	first          error
}

func (w *follower) fail(path string, err error) {
	if w.first == nil {
		w.first = &WalkError{Path: path, Err: err}
	}
}

// inside reports whether real (a path with no symlink left) is the root's real path or under it.
func (w *follower) inside(real string) bool {
	return real == w.realRoot || strings.HasPrefix(real, strings.TrimSuffix(w.realRoot, "/")+"/")
}

// spelled is real (inside the root) spelled from the root as given, the way the walk spells a path.
func (w *follower) spelled(real string) string {
	rel := strings.TrimPrefix(strings.TrimPrefix(real, w.realRoot), "/")
	if rel == "" {
		return w.root
	}
	return strings.TrimSuffix(w.root, "/") + "/" + rel
}

func (w *follower) walk(dir string) {
	real, err := filepath.EvalSymlinks(dir)
	if err != nil {
		w.fail(dir, err)
		return
	}
	if w.seen[real] {
		return
	}
	w.seen[real] = true
	entries, err := os.ReadDir(dir)
	if err != nil {
		w.fail(dir, err)
		return
	}
	prefix := dir + "/"
	if strings.HasSuffix(dir, "/") {
		prefix = dir
	}
	for _, e := range entries {
		w.entry(e.Name(), prefix+e.Name(), e.Type())
	}
}

// entry handles one directory entry: name is its name, full its path as the walk spells it.
func (w *follower) entry(name, full string, mode os.FileMode) {
	selected := w.sel.selects(name)
	pruneAs := []string{name, full}
	if mode&os.ModeSymlink != 0 {
		st, err := os.Stat(full)
		if err != nil {
			if selected {
				// Mutation: WF-dangling -- ignoring a dangling link whose name is selected reads a template nobody could open as absent
				w.fail(full, err)
			}
			return
		}
		mode = st.Mode().Type()
		real, err := filepath.EvalSymlinks(full)
		if err != nil {
			w.fail(full, err)
			return
		}
		if !w.inside(real) {
			// Mutation: WF-outside -- following a link out of the root reads a sibling module, or the whole file system
			if mode.IsDir() || selected {
				w.outside = append(w.outside, full)
			}
			return
		}
		pruneAs = append(pruneAs, filepath.Base(real), w.spelled(real))
	}
	switch {
	case mode.IsDir():
		// Mutation: WF-prune-target -- pruning a link by its own name alone lets out -> build re-expose a build output
		for i := 0; i < len(pruneAs); i += 2 {
			if w.sel.prunes(pruneAs[i], pruneAs[i+1]) {
				return
			}
		}
		w.walk(full)
	case !selected:
	case !mode.IsRegular():
		// Mutation: WF-not-regular -- skipping a selected FIFO prints "nothing to check" over a template nobody read
		w.fail(full, ErrNotRegular)
	case len(w.sel.Paths) == 0 || matchAny(w.sel.Paths, full, true):
		w.file(full)
	}
}

// file lists full once per real file: a second spelling of a file already listed is dropped.
func (w *follower) file(full string) {
	real, err := filepath.EvalSymlinks(full)
	if err != nil {
		w.fail(full, err)
		return
	}
	if !w.seen[real] {
		w.seen[real] = true
		w.files = append(w.files, full)
	}
}
