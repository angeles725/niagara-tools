package preflight

import (
	"os"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// jarLock warns about module jars a process holds open (a running station), from ONE `lsof -Fn`
// listing matched against the jar set: a per-jar lsof (or a per-jar scan of the listing) took minutes
// on a 1013-jar home [ev: retro r10-fork-per-line-again Δ1]. lsof stays the source: /proc/<pid>/fd alone misses mapped files and
// other users' processes lsof may see. Never a false PASS: SKIP on a 9p/drvfs niagara_home (lsof
// cannot see a Windows-side lock), without lsof, or on an empty listing (a working lsof always lists
// its own files). A match is a whole `n<path>` line, by path string, not device+inode.
func (r *run) jarLock() {
	if r.win {
		return
	}
	if t := r.fsType(r.nh); t == "9p" || t == "drvfs" {
		r.row("SKIP", "jar-lock", "niagara_home is on a 9p/drvfs mount (WSL) — lsof cannot see a Windows-side lock here")
		return
	}
	lsof, ok := lookPath("lsof", r.s.Env("PATH"))
	if !ok {
		r.row("SKIP", "jar-lock", "lsof not in PATH — install lsof to enable jar-lock detection")
		return
	}
	listing := strings.TrimRight(string(r.h.output(lsof, []string{"-Fn"}, false)), "\n")
	held := map[string]bool{}
	for _, ln := range strings.Split(listing, "\n") {
		held[ln] = true
	}
	locked := ""
	entries, _ := os.ReadDir(r.nh + "/modules")
	for _, e := range entries {
		jar := r.nh + "/modules/" + e.Name()
		if strings.HasSuffix(e.Name(), ".jar") && !strings.HasPrefix(e.Name(), ".") && lint.IsFile(jar) && held["n"+jar] {
			locked += " " + e.Name()
		}
	}
	switch {
	case locked != "":
		r.row("WARN", "jar-lock", "jars held by a process (station running?):"+locked)
	case listing == "":
		r.row("SKIP", "jar-lock", "lsof returned no open-file listing (failed or no permission) — jar locks not checked")
	default:
		r.row("PASS", "jar-lock", "no locked jars under "+r.nh+"/modules/")
	}
}

// fsType is lib/fs-type.sh fs_type_for: the filesystem type of path from the longest mountpoint string
// prefix in $N4_FSTYPE_MOUNTS_FILE (a /proc/mounts-format file tests pass), else `df -T -- <path>`.
func (r *run) fsType(path string) string {
	mf := r.s.Env("N4_FSTYPE_MOUNTS_FILE")
	if mf == "" || !lint.IsFile(mf) {
		df, ok := lookPath("df", r.s.Env("PATH")) // found as bash finds it; none prints nothing
		if !ok {
			return ""
		}
		return field(line(r.h.output(df, []string{"-T", "--", path}, false), 1), 1)
	}
	b, _ := os.ReadFile(mf)
	best, typ := 0, ""
	for _, ln := range strings.Split(string(b), "\n") {
		if mp := field(ln, 1); mp != "" && strings.HasPrefix(path, mp) && len(mp) > best {
			best, typ = len(mp), field(ln, 2)
		}
	}
	return typ
}

// line is line i (0-based) of out, "" past its end.
func line(out []byte, i int) string {
	if ls := strings.Split(string(out), "\n"); i < len(ls) {
		return ls[i]
	}
	return ""
}

// field is awk's $(i+1) under the default FS (runs of blanks and tabs, leading ones skipped).
func field(s string, i int) string {
	if f := strings.FieldsFunc(s, func(c rune) bool { return c == ' ' || c == '\t' }); i < len(f) {
		return f[i]
	}
	return ""
}

// lookPath is bash's `command -v name` over path: the first executable regular file dir/name (an
// empty dir is "."), else the first existing non-directory one, which bash also returns (it then
// fails to run: an empty listing, SKIP).
func lookPath(name, path string) (string, bool) {
	fallback := ""
	for _, dir := range strings.Split(path, ":") {
		if dir == "" {
			dir = "."
		}
		p := dir + "/" + name
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			if syscall.Access(p, 1) == nil {
				return p, true
			}
			if fallback == "" {
				fallback = p
			}
		}
	}
	return fallback, fallback != ""
}
