// Package fstype is the one Go detector of the filesystem type backing a path (toolbelt/lib/fs-type.sh
// fs_type_for), for the WSL 9p/drvfs gates of preflight (jar-lock) and build-module (build location).
// TestMatchesTheShellLib keeps it equal to the bash lib while that lib exists.
package fstype

import (
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// MountsEnv names a /proc/mounts-format file (device mountpoint fstype ...) that replaces `df -T`:
// tests simulate a 9p/drvfs mount with it.
const MountsEnv = "N4_FSTYPE_MOUNTS_FILE"

// For is the filesystem type of path, "" when it cannot be told: from the longest mountpoint string
// prefix in mountsFile (the value of MountsEnv) when that names a file, else from the second column of
// line 2 of what df returns, the stdout of `df -T -- <path>` (nothing when df is missing or fails).
func For(path, mountsFile string, df func() []byte) string {
	if mountsFile == "" || !lint.IsFile(mountsFile) {
		return field(line(df(), 1), 1)
	}
	b, _ := os.ReadFile(mountsFile)
	best, typ := 0, ""
	for _, ln := range strings.Split(string(b), "\n") {
		if mp := field(ln, 1); mp != "" && strings.HasPrefix(path, mp) && len(mp) > best {
			best, typ = len(mp), field(ln, 2)
		}
	}
	return typ
}

// IsWindowsMount reports whether typ is a WSL 9p or drvfs mount (a Windows-side path under /mnt/<drive>).
func IsWindowsMount(typ string) bool { return typ == "9p" || typ == "drvfs" }

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
