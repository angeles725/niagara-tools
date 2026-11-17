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

// resolve sets srcAbs and dirAbs. A mirror dir that exists is resolved itself (its parent, when it is not
// a directory); one that does not is first judged by guard A where it would lead (its longest existing
// ancestor resolved, the rest by its text), and only then gets its parent made and resolved, with its own
// name appended. So a refused mirror dir never leaves a directory behind, and one that passes is made where
// guard A judged it (at), never along its spelling: "<src>/new/../../out" makes nothing inside <src>.
func (m *mirror) resolve() int {
	var err error
	if m.srcAbs, err = physical(m.src); err != nil {
		return m.fault(err)
	}
	m.at = m.dir
	st, statErr := os.Stat(m.dir) // test -e: a dangling symlink is not there
	m.exists = statErr == nil
	if m.exists && st.IsDir() {
		if m.dirAbs, err = physical(m.dir); err != nil {
			return m.fault(err)
		}
		return proceed
	}
	if !m.exists {
		if code := m.guardAhead(); code != proceed {
			return code
		}
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

// guardAhead is guard A on a mirror dir that is not there yet, before anything is created; the path it
// judged becomes at. A mirror dir that cannot be resolved at all (a link loop, a directory that cannot be
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
