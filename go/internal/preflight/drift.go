package preflight

import (
	"os"
	"path"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/findwalk"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// versionDrift warns when a sibling tree declares a higher defaultModuleVersion("X.Y.Z") than the gradle
// root: the siblings of its parent, then of the next two ancestors (3 levels), each tree's version being
// the first one in its *.gradle.kts files down to depth 3; `<repo>-worktrees/` checkouts and the gradle root's
// own ancestors are skipped. The walk stays inside the tree of the gradle root: it stops at a shared directory
// (sharedDir) and skips, without a line, a directory it cannot read [ev: retro build-kit-bench-session Δ16].
// WARN only [ev: retro B832].
func (r *run) versionDrift() {
	if r.win {
		return
	}
	mine := r.treeVersion(r.gr)
	if mine == "" {
		return
	}
	parent := lint.Dirname(r.gr)
	for lvl := 0; lvl < 3 && lint.IsDir(parent) && !sharedDir(parent); lvl++ {
		for _, sib := range globDirs(parent) {
			if sib == r.gr || notASibling(sib, r.gr) {
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

// sharedDir says whether dir holds other people's trees, not the siblings of a module: the filesystem root, the
// temporary directories and the home directory. The drift walk does not look at their entries.
func sharedDir(dir string) bool {
	dir = strings.TrimRight(dir, "/")
	if dir == "" {
		return true // the filesystem root
	}
	home, _ := os.UserHomeDir()
	for _, shared := range []string{"/tmp", "/var/tmp", os.TempDir(), home} {
		if shared != "" && dir == strings.TrimRight(shared, "/") {
			return true
		}
	}
	return false
}

// notASibling says whether sib is another checkout of the same repository (a `<repo>-worktrees/` directory) or
// an ancestor of the gradle root (the repository root holding it): neither is a sibling module, so a newer
// version there is no drift [ev: retro kit-followups-rare-cases Δ3].
func notASibling(sib, gradleRoot string) bool {
	return strings.HasSuffix(sib, "-worktrees") || strings.HasPrefix(gradleRoot, sib+"/")
}

// treeVersion is the first quoted X.Y.Z on a line holding defaultModuleVersion, over every *.gradle.kts
// (any file type, symlinks read through) that `find -H <dir> -maxdepth 3` lists, in its order. A
// directory find cannot read is reported on stderr as find words it, and the walk goes on.
func (r *run) treeVersion(dir string) string {
	var kts []string
	findwalk.Walk(dir, true, 3, func(e findwalk.Entry) bool {
		if ok, _ := path.Match("*.gradle.kts", e.Name); ok {
			kts = append(kts, e.Path)
		}
		return true
	}, func(string, error) {}) // a directory it cannot read is skipped without a line
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
