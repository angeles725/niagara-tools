package mirrorhome

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/orientguard"
)

// physical is `cd <dir> && pwd -P`: the directory with every symlink resolved. It fails when dir is not
// a directory that can be entered.
func physical(dir string) (string, error) {
	abs := dir
	if !strings.HasPrefix(dir, "/") {
		cwd, err := syscall.Getwd()
		if err != nil {
			return "", &fs.PathError{Op: "cd", Path: dir, Err: err}
		}
		abs = cwd + "/" + dir
	}
	if st, err := os.Stat(abs); err != nil {
		return "", &fs.PathError{Op: "cd", Path: dir, Err: pathErr(err)}
	} else if !st.IsDir() {
		return "", &fs.PathError{Op: "cd", Path: dir, Err: syscall.ENOTDIR}
	}
	if err := syscall.Access(abs, 1); err != nil { // X_OK: cd needs search permission
		return "", &fs.PathError{Op: "cd", Path: dir, Err: err}
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", &fs.PathError{Op: "cd", Path: dir, Err: pathErr(err)}
	}
	return resolved, nil
}

// pathErr is the cause inside a *fs.PathError (err itself otherwise).
func pathErr(err error) error {
	if pe, ok := err.(*fs.PathError); ok {
		return pe.Err
	}
	return err
}

// resolve sets srcAbs and at: the mirror dir as guard A judges it, every symlink followed where components
// exist and the rest applied by its text (orientguard.Reachable, `realpath -m`). Every later step looks at
// and writes to at, never the spelling, so "ml/../o" and "missing/../dir" are the directories they lead to.
// Nothing is created here: build makes the mirror once every guard has passed on at. A mirror dir that
// cannot be resolved at all (a link loop, a directory that cannot be searched) is refused.
func (m *mirror) resolve() int {
	var err error
	if m.srcAbs, err = physical(m.src); err != nil {
		return m.fault(err)
	}
	cwd, err := syscall.Getwd()
	if err != nil {
		return m.fault(&fs.PathError{Op: "getcwd", Path: ".", Err: err})
	}
	at, ok := orientguard.Reachable(m.dir, cwd)
	if !ok {
		return m.stop(ExitRefused, msgUnresolved, m.dir)
	}
	m.at = at
	_, err = os.Stat(m.at)
	m.exists = err == nil
	return proceed
}

// linkAt reports whether the mirror dir's last component is a symlink, looked at where its parent leads:
// final slashes and final "/." are dropped first, so "link/", "link/." and "missing/../link" are the link.
// A last component ".." names a directory, never a link.
func (m *mirror) linkAt() bool {
	name := m.dir
	for {
		trimmed := strings.TrimSuffix(strings.TrimRight(name, "/"), "/.")
		if trimmed == name || trimmed == "" {
			break
		}
		name = trimmed
	}
	base := lint.Basename(name)
	if base == ".." || base == "." || base == "/" {
		return false
	}
	cwd, err := syscall.Getwd()
	if err != nil {
		return false
	}
	parent, ok := orientguard.Reachable(lint.Dirname(name), cwd)
	if !ok {
		return false
	}
	st, err := os.Lstat(strings.TrimSuffix(parent, "/") + "/" + base)
	return err == nil && st.Mode()&os.ModeSymlink != 0
}
