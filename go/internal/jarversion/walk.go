package jarversion

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
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
		w.err = "find: " + findQuote(path) + ": " + Reason(err)
	}
}

// join appends name to dir like find: no second slash after a root that already ends in one.
func join(dir, name string) string {
	if strings.HasSuffix(dir, "/") {
		return dir + name
	}
	return dir + "/" + name
}

// strerror is the C-locale strerror text of errno (Go's errno strings are glibc's, lower-cased).
func strerror(errno syscall.Errno) string {
	s := errno.Error()
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		s = string(s[0]-'a'+'A') + s[1:]
	}
	return s
}

// findQuote quotes a path as GNU find does in an LC_ALL=C error message (gnulib locale_quoting_style):
// single quotes; a backslash or a single quote is backslash-escaped, the C control escapes are used for
// \a\b\f\n\r\t\v, and any other byte outside printable ASCII is a three-digit octal escape.
func findQuote(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\\' || c == '\'':
			b.WriteByte('\\')
			b.WriteByte(c)
		case strings.IndexByte("\a\b\f\n\r\t\v", c) >= 0:
			b.WriteByte('\\')
			b.WriteByte("abfnrtv"[strings.IndexByte("\a\b\f\n\r\t\v", c)])
		case c < 0x20 || c >= 0x7f:
			fmt.Fprintf(&b, "\\%03o", c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteByte('\'')
	return b.String()
}
