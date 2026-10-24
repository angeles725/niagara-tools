package preflight

import (
	"os"
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// versionDrift warns when a sibling tree declares a higher defaultModuleVersion("X.Y.Z") than the gradle
// root: the siblings of its parent, then of the next two ancestors (3 levels), each tree's version being
// the first one in its *.gradle.kts files down to depth 3. WARN only [ev: retro B832].
func (r *run) versionDrift() {
	if r.win {
		return
	}
	mine := r.treeVersion(r.gr)
	if mine == "" {
		return
	}
	parent := lint.Dirname(r.gr)
	for lvl := 0; lvl < 3 && lint.IsDir(parent); lvl++ {
		for _, sib := range globDirs(parent) {
			if sib == r.gr {
				continue
			}
			if v := r.treeVersion(sib); v != "" && versionLess(mine, v) {
				r.row("WARN", "version-drift", r.gr+" version "+mine+" < sibling "+sib+" version "+v)
			}
		}
		next := lint.Dirname(parent)
		if next == parent {
			break
		}
		parent = next
	}
}

// treeVersion is the first quoted X.Y.Z on a line holding defaultModuleVersion, over every *.gradle.kts
// (any file type, symlinks read through) that `find -H <dir> -maxdepth 3` lists, in its order. A
// directory find cannot read is reported on stderr as find words it, and the walk goes on.
func (r *run) treeVersion(dir string) string {
	var kts []string
	findWalk(dir, true, 3, func(e entry) bool {
		if ok, _ := path.Match("*.gradle.kts", e.name); ok {
			kts = append(kts, e.path)
		}
		return true
	}, func(p string, err error) {
		r.s.Err.Write([]byte("find: " + strerror.FindQuote(p) + ": " + strerror.Text(err) + "\n"))
	})
	for _, f := range kts {
		b, _ := os.ReadFile(f) // a directory or an unreadable file holds no line, as for grep
		for _, ln := range strings.Split(grepText(b), "\n") {
			if v := quotedVersion.FindString(ln); v != "" && strings.Contains(ln, "defaultModuleVersion") {
				return strings.Trim(v, `"`)
			}
		}
	}
	return ""
}

// versionLess is `sort -V` placing a strictly before b, for two X.Y.Z strings: numeric per component,
// equal values ordered bytewise (sort's last-resort compare).
func versionLess(a, b string) bool {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range pa {
		x, y := strings.TrimLeft(pa[i], "0"), strings.TrimLeft(pb[i], "0")
		if len(x) != len(y) {
			return len(x) < len(y)
		}
		if x != y {
			return x < y
		}
	}
	return a < b
}
