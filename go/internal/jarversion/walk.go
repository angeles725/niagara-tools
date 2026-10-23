package jarversion

import (
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// ListJars lists the regular files named *.jar under root the way lib/scan-files.sh does with
// `find -H <root> -mindepth 1 \( -type d \( -name '.*' -o -name build \) -prune \) -o -type f -name '*.jar' -print`
// (LC_ALL=C sort): every sub-directory named build or starting with a dot is pruned (the root
// itself never is), a symlinked root is followed (issue #329) but symlinks below it are neither listed
// nor followed, and paths are built from root as given. walkErr is the first error find would have printed
// ("find: '<dir>': <reason>"), "" for a complete walk; the files it did list are returned either way.
func ListJars(root string) (jars []string, walkErr string) {
	w := walker{}
	if st, err := os.Stat(root); err != nil {
		w.fail(root, err)
	} else if st.IsDir() {
		w.dir(root)
	}
	sort.Strings(w.jars)
	return w.jars, w.err
}

type walker struct {
	jars []string
	err  string
}

// dir visits one directory pre-order in directory (readdir) order, as find does.
func (w *walker) dir(path string) {
	f, err := os.Open(path)
	if err != nil {
		w.fail(path, err)
		return
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	for _, e := range entries {
		name, child := e.Name(), join(path, e.Name())
		switch t := e.Type(); {
		case t.IsDir():
			if !strings.HasPrefix(name, ".") && name != "build" {
				w.dir(child)
			}
		case t.IsRegular() && strings.HasSuffix(name, ".jar"):
			w.jars = append(w.jars, child)
		}
	}
	if err != nil {
		w.fail(path, err)
	}
}

// fail records the first walk error in find's wording.
func (w *walker) fail(path string, err error) {
	if w.err == "" {
		w.err = "find: " + strerror.FindQuote(path) + ": " + Reason(err)
	}
}

// join appends name to dir like find: no second slash after a root that already ends in one.
func join(dir, name string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}
