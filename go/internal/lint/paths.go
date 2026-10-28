package lint

import (
	"os"
	"path/filepath"
	"strings"
)

// The shell path tests a RootRule reads its root with (it walks nothing through the runner): POSIX
// dirname(1) and basename(1) on the path as given, test -f and test -d (symlinks followed).

// Dirname is dirname(1): trailing slashes dropped, "." for a name without a slash, "/" for the root.
func Dirname(p string) string {
	p = strings.TrimRight(p, "/")
	if p == "" {
		return "/"
	}
	i := strings.LastIndexByte(p, '/')
	if i < 0 {
		return "."
	}
	if p = strings.TrimRight(p[:i], "/"); p == "" {
		return "/"
	}
	return p
}

// Basename is basename(1): the last component, trailing slashes dropped ("/" for the root).
func Basename(p string) string {
	if p = strings.TrimRight(p, "/"); p == "" {
		return "/"
	}
	return p[strings.LastIndexByte(p, '/')+1:]
}

// IsFile is test -f: a regular file, symlinks followed.
func IsFile(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.Mode().IsRegular()
}

// IsDir is test -d: a directory, symlinks followed.
func IsDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// MapFile is the module map at a module root (types/structure.md § Module map).
const MapFile = "MODULE-MAP.md"

// Modularity is a module's modularity mode, declared in its module map's header.
type Modularity int

const (
	ModularityNone   Modularity = iota // no module map, or a map whose header is not valid
	ModularityLegacy                   // modularity: legacy (with a legacy-reason: line): an existing module, WARN rows
	ModularityStrict                   // modularity: strict: a new (scaffolded) module, FAIL rows
)

func (m Modularity) String() string {
	switch m {
	case ModularityStrict:
		return "strict"
	case ModularityLegacy:
		return "legacy"
	}
	return "none"
}

// MapHeader reads a module map's header: the lines before its first table row (a line starting with "|"),
// outside ``` fences. The first "modularity:" line decides (a trailing "# comment" is dropped): strict, or
// legacy with a non-empty "legacy-reason:" line anywhere in the header. It returns the mode and the 1-based
// line of that modularity line; for a header that is not valid, ModularityNone, that line (0 when there is
// none) and the problem, worded for an MM2 row of lint-module-map.
func MapHeader(lines []string) (Modularity, int, string) {
	var value, reason string
	at, fenced, hasReason := 0, false, false
	for i, raw := range lines {
		l := strings.TrimSpace(raw)
		if strings.HasPrefix(l, "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if strings.HasPrefix(l, "|") {
			break
		}
		if v, ok := strings.CutPrefix(l, "modularity:"); ok && at == 0 {
			if c := strings.IndexByte(v, '#'); c >= 0 {
				v = v[:c]
			}
			value, at = strings.TrimSpace(v), i+1
		}
		if v, ok := strings.CutPrefix(l, "legacy-reason:"); ok && !hasReason {
			reason, hasReason = strings.TrimSpace(v), true
		}
	}
	switch {
	case at == 0:
		return ModularityNone, 0, "no modularity: line before the table (want modularity: strict or modularity: legacy)"
	case value == "strict":
		return ModularityStrict, at, ""
	case value != "legacy":
		return ModularityNone, at, "modularity: " + value + " is not strict or legacy"
	case reason == "":
		return ModularityNone, at, "modularity: legacy needs a non-empty legacy-reason: line"
	}
	return ModularityLegacy, at, ""
}

// ModuleMode is the modularity mode that applies to path (a file or a directory), for a per-file lint that
// picks its severity by mode: it walks up from path (made absolute) to the first directory holding a
// MODULE-MAP.md regular file and returns that map's header mode. The walk stops after a directory that holds
// .git (a directory, or a worktree's .git file), so a map outside the repository is never read, and at the
// filesystem root. No map, an unreadable map or an invalid header is ModularityNone.
func ModuleMode(path string) Modularity {
	dir, err := filepath.Abs(path)
	if err != nil {
		return ModularityNone
	}
	if !IsDir(dir) {
		dir = Dirname(dir)
	}
	for {
		if m := dir + "/" + MapFile; IsFile(m) {
			b, err := os.ReadFile(m)
			if err != nil {
				return ModularityNone
			}
			mode, _, _ := MapHeader(strings.Split(string(b), "\n"))
			return mode
		}
		if _, err := os.Stat(dir + "/.git"); err == nil || dir == "/" {
			return ModularityNone
		}
		dir = Dirname(dir)
	}
}
