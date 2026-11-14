package preflight

import (
	"os"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/fstype"
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
	if fstype.IsWindowsMount(r.fsType(r.nh)) {
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

// fsType is the filesystem type of path (fstype.For); df is found on PATH as bash finds it.
func (r *run) fsType(path string) string {
	return fstype.For(path, r.s.Env(fstype.MountsEnv), func() []byte {
		df, ok := lookPath("df", r.s.Env("PATH"))
		if !ok {
			return nil
		}
		return r.h.output(df, []string{"-T", "--", path}, false)
	})
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
