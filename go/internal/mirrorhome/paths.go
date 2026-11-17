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

// resolve sets srcAbs, at and dirAbs. A mirror dir its spelling does not reach (a missing directory on the
// way, a dangling link) is first judged by guard A where it leads (its longest existing ancestor resolved,
// the rest by its text), and from then on at, the path judged, is the one looked at and written: so
// "<src>/new/../../out" makes nothing inside <src>, and "missing/../dir" meets guard B at dir. A dangling
// link is refused before anything is made. The mirror dir at at is then resolved itself when it is a
// directory, else its parent is made (when it is not there) and resolved, with its own name appended.
func (m *mirror) resolve() int {
	var err error
	if m.srcAbs, err = physical(m.src); err != nil {
		return m.fault(err)
	}
	m.at = m.dir
	if _, err := os.Stat(m.dir); err != nil { // test -e: a dangling symlink is not there
		if code := m.guardAhead(); code != proceed {
			return code
		}
		if m.isLink() {
			return m.stop(ExitRefused, msgLink, m.dir)
		}
	}
	st, statErr := os.Stat(m.at)
	m.exists = statErr == nil
	if m.exists && st.IsDir() {
		if m.dirAbs, err = physical(m.at); err != nil {
			return m.fault(err)
		}
		return proceed
	}
	parent := lint.Dirname(m.at)
	if !m.exists {
		if err := os.MkdirAll(parent, 0o777); err != nil {
			return m.fault(err)
		}
	}
	if parent, err = physical(parent); err != nil {
		return m.fault(err)
	}
	m.dirAbs = parent + "/" + lint.Basename(m.at)
	return proceed
}

// guardAhead is guard A on a mirror dir its spelling does not reach, before anything is created; the path
// it judged becomes at. A mirror dir that cannot be resolved at all (a link loop, a directory that cannot be
// searched) is refused.
func (m *mirror) guardAhead() int {
	cwd, err := syscall.Getwd()
	if err != nil {
		return m.fault(&fs.PathError{Op: "getcwd", Path: ".", Err: err})
	}
	leads, ok := orientguard.Reachable(m.dir, cwd)
	if !ok {
		return m.stop(ExitRefused, msgUnresolved, m.dir)
	}
	if refusal := sourceRefusal(m.srcAbs, leads); refusal != "" {
		return m.stop(ExitRefused, refusal, m.dir)
	}
	m.at = leads
	return proceed
}
