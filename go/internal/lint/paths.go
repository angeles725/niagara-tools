package lint

import (
	"os"
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
