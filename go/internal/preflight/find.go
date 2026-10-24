package preflight

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// entry is one path a find walk visits, spelled as GNU find prints it.
type entry struct {
	path, name string
	dir        bool // a directory (a symlink below the root is never one: it is not followed)
	depth      int
}

// findWalk walks root as GNU find does: the root first (followed when it is a symlink and followRoot,
// find -H), then depth-first, children spelled root + "/" + name, but in bytewise name order where find
// takes readdir order: the bash version-drift took the first version in readdir order, which differs
// between filesystems (ext4 hash order), so the same tree drifted on one host and not on another. visit
// returns whether to enter a directory; maxDepth bounds the entered depth (find -maxdepth). An
// unreadable directory goes to onErr (nil: ignored) and the walk goes on.
func findWalk(root string, followRoot bool, maxDepth int, visit func(entry) bool, onErr func(path string, err error)) {
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
	var walk func(e entry)
	walk = func(e entry) {
		if !visit(e) || !e.dir || e.depth >= maxDepth {
			return
		}
		kids, err := os.ReadDir(e.path)
		if err != nil && onErr != nil {
			onErr(e.path, err)
		}
		prefix := e.path + "/"
		if strings.HasSuffix(e.path, "/") {
			prefix = e.path
		}
		for _, k := range kids {
			walk(entry{prefix + k.Name(), k.Name(), k.IsDir(), e.depth + 1})
		}
	}
	walk(entry{root, lint.Basename(root), st.IsDir(), 0})
}
