package stationsnapshot

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// errText is the C-library message of err ("Permission denied"), as bash and coreutils print it.
func errText(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		err = pe.Err
	}
	return strerror.Text(err)
}

// findQuote and mkdirQuote spell a path as GNU find and mkdir quote it in the C locale.
func findQuote(p string) string  { return strerror.FindQuote(p) }
func mkdirQuote(p string) string { return strerror.FindQuote(p) }

// cpQuote spells a path as GNU cp quotes it in the C locale ('...' for a plain name; other names fall back to
// the locale style, which differs from cp's shell-escape style only for quotes and non-printable bytes).
func cpQuote(p string) string { return strerror.FindQuote(p) }

func isDir(p string) bool     { st, err := os.Stat(p); return err == nil && st.IsDir() }
func isRegular(p string) bool { st, err := os.Stat(p); return err == nil && st.Mode().IsRegular() }

// searchable is the error chdir(p) would give: none for a directory the user may enter.
func searchable(p string) error {
	st, err := os.Stat(p)
	switch {
	case err != nil:
		return err
	case !st.IsDir():
		return syscall.ENOTDIR
	}
	return syscall.Access(p, 1)
}

// logicalCd is the bash `cd dir && pwd` (cd -L): dir joined to the logical cwd and cleaned lexically; when
// that path cannot be entered, the physical dir, as bash falls back to it.
func logicalCd(dir string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	p := dir
	if !strings.HasPrefix(dir, "/") {
		p = wd + "/" + dir
	}
	p = filepath.Clean(p)
	if searchable(p) == nil {
		return p, nil
	}
	if err := searchable(dir); err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(dir)
}

// basename and dirname are the POSIX utilities.
func basename(s string) string {
	t := strings.TrimRight(s, "/")
	if t == "" {
		if s == "" {
			return ""
		}
		return "/"
	}
	return t[strings.LastIndex(t, "/")+1:]
}

func dirname(s string) string {
	t := strings.TrimRight(s, "/")
	if t == "" {
		if s == "" {
			return "."
		}
		return "/"
	}
	i := strings.LastIndex(t, "/")
	if i < 0 {
		return "."
	}
	if t = strings.TrimRight(t[:i], "/"); t == "" {
		return "/"
	}
	return t
}

// mkdirP is GNU `mkdir -p dir`: each missing component made in turn, entering each ancestor; on failure it
// prints mkdir's message (naming the ancestor that could not be made or entered, or dir itself) and returns
// false.
func mkdirP(dir string, errOut io.Writer) bool {
	fail := func(p string, err error) bool {
		fmt.Fprintf(errOut, "mkdir: cannot create directory %s: %s\n", mkdirQuote(p), errText(err))
		return false
	}
	if dir == "" {
		return fail(dir, syscall.ENOENT)
	}
	parts := strings.Split(dir, "/")
	last := len(parts) - 1
	for last > 0 && parts[last] == "" {
		last--
	}
	for i := 0; i < last; i++ {
		if parts[i] == "" || parts[i] == "." {
			continue
		}
		p := strings.Join(parts[:i+1], "/")
		err := os.Mkdir(p, 0o777)
		if err == nil || searchable(p) == nil {
			continue
		}
		if errors.Is(err, fs.ErrExist) {
			err = searchable(p)
		}
		return fail(p, err)
	}
	if err := os.Mkdir(dir, 0o777); err != nil && !isDir(dir) {
		return fail(dir, err)
	}
	return true
}

// onPath is bash `command -v name` for a command found only on PATH: a regular file named name in a PATH
// directory (an empty entry is the cwd), executable or not.
func onPath(name, path string) bool {
	for _, dir := range strings.Split(path, ":") {
		if dir == "" {
			dir = "."
		}
		if isRegular(dir + "/" + name) {
			return true
		}
	}
	return false
}
