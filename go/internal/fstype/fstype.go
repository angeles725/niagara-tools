// Package fstype is the one detector of the filesystem type backing a path, for the WSL 9p/drvfs gates of
// preflight (jar-lock) and build-module (build location). It replaced the bash lib toolbelt/lib/fs-type.sh
// (fs_type_for, is_9p_or_drvfs); its tests keep the answers that lib gave.
package fstype

import (
	"os"
	"path/filepath"
	"strconv"
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

// ProcMounts is the kernel's mount table, the source of OfPath without a MountsEnv file.
const ProcMounts = "/proc/mounts"

// OfPath is the filesystem type of path read from a mount table (mountsFile, else ProcMounts) with no
// helper command: the type of the mount whose mountpoint is the longest whole-component prefix of the
// path, spelled as the kernel resolves it (symlinks followed, a relative path made absolute). "" when
// the table cannot be read or names no mount for it; of two entries of one mountpoint the last (the topmost,
// as df answers) wins. Unlike For, which keeps the bash lib's string
// prefix for preflight, "/mnt/cc" is not under a mount at "/mnt/c".
func OfPath(path, mountsFile string) string {
	if mountsFile == "" || !lint.IsFile(mountsFile) {
		mountsFile = ProcMounts
	}
	table, err := os.ReadFile(mountsFile)
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	best, typ := -1, ""
	for _, ln := range strings.Split(string(table), "\n") {
		mp := unescapeMount(field(ln, 1))
		if mp == "" || !under(path, mp) || len(mp) < best {
			continue
		}
		best, typ = len(mp), field(ln, 2)
	}
	return typ
}

// under reports whether path is mp or lies below it, by whole components.
func under(path, mp string) bool {
	return mp == "/" || path == mp || strings.HasPrefix(path, mp+"/")
}

// unescapeMount undoes the \040-style octal escapes the kernel writes into a mount table (a blank in a
// mountpoint is \040).
func unescapeMount(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+4 <= len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
