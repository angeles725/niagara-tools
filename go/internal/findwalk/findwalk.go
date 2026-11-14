// Package findwalk walks a tree as GNU find does, for the tools ported from a bash `find` call
// (preflight, build-module): one walk, so two tools never disagree on what a tree holds.
package findwalk

import (
	"os"
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// NoLimit is the maxDepth of a walk without -maxdepth.
const NoLimit = 1 << 30

// Entry is one path a walk visits, spelled as GNU find prints it.
type Entry struct {
	Path, Name string
	Dir        bool // a directory (a symlink below the root is never one: it is not followed)
	Depth      int
}

// Walk walks root as GNU find does: the root first (followed when it is a symlink and followRoot,
// find -H), then depth-first, children spelled root + "/" + name, but in bytewise name order where find
// takes readdir order: the bash version-drift took the first version in readdir order, which differs
// between filesystems (ext4 hash order), so the same tree drifted on one host and not on another. visit
// returns whether to enter a directory; maxDepth bounds the entered depth (find -maxdepth). An
// unreadable directory goes to onErr (nil: ignored) and the walk goes on.
func Walk(root string, followRoot bool, maxDepth int, visit func(Entry) bool, onErr func(path string, err error)) {
	stat := os.Lstat
	if followRoot {
		stat = os.Stat
	}
	st, err := stat(root)
	if err != nil {
		if onErr != nil {
			onErr(root, err)
		}
		return
	}
	var walk func(e Entry)
	walk = func(e Entry) {
		if !visit(e) || !e.Dir || e.Depth >= maxDepth {
			return
		}
		kids, err := os.ReadDir(e.Path)
		if err != nil && onErr != nil {
			onErr(e.Path, err)
		}
		prefix := e.Path + "/"
		if strings.HasSuffix(e.Path, "/") {
			prefix = e.Path
		}
		for _, k := range kids {
			walk(Entry{prefix + k.Name(), k.Name(), k.IsDir(), e.Depth + 1})
		}
	}
	walk(Entry{root, lint.Basename(root), st.IsDir(), 0})
}

// HasDir is `find <root> -type d -name <name> | grep -q .`: a directory under root, root included, whose
// name matches the pattern name.
func HasDir(root, name string) bool { return HasDirWithin(root, name, NoLimit) }

// HasDirWithin is HasDir bounded by `find -maxdepth <maxDepth>`.
func HasDirWithin(root, name string, maxDepth int) bool {
	found := false
	Walk(root, false, maxDepth, func(e Entry) bool {
		ok, _ := path.Match(name, e.Name)
		found = found || (e.Dir && ok)
		return !found
	}, nil)
	return found
}
