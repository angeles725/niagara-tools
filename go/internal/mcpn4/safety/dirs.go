package safety

import (
	"errors"
	"io/fs"
	"os"
	"strings"
)

// IsDir is os.path.isdir: false on any stat error.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// MakeDirs is os.makedirs(name, mode, exist_ok=True): missing parents get 0777 (less the
// umask), name gets mode.
func MakeDirs(name string, mode fs.FileMode) error {
	head := Dirname(name)
	if tail := name[len(head):]; strings.Trim(tail, "/") != "" && head != "" && head != name {
		if _, err := os.Lstat(head); err != nil {
			if err := MakeDirs(head, 0o777); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
		}
	}
	if err := os.Mkdir(name, mode); err != nil && !IsDir(name) {
		return err
	}
	return nil
}

// PrivateDir creates dir 0700 when it is not a directory yet, as
//
//	if not os.path.isdir(d): os.makedirs(d, mode=0o700, exist_ok=True); os.chmod(d, 0o700)
//
// does: only a directory this call creates is chmodded, an existing one keeps its mode.
func PrivateDir(dir string) error {
	if IsDir(dir) {
		return nil
	}
	if err := MakeDirs(dir, 0o700); err != nil {
		return err
	}
	return os.Chmod(dir, 0o700)
}
