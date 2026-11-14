package installtools

import (
	"errors"
	"io/fs"
	"os"
	"syscall"
)

// absent reports whether a stat error means "there is no such file" (a missing name, or a path through
// something that is not a directory) and not "could not look".
func absent(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR)
}

// IsFile is a shell `[ -f path ]` (symlinks followed) with the third answer the shell hides: err is
// non-nil when the path could not be examined for another reason than its absence.
func IsFile(path string) (bool, error) {
	st, err := os.Stat(path)
	switch {
	case err == nil:
		return st.Mode().IsRegular(), nil
	case absent(err):
		return false, nil
	}
	return false, err
}

// Exists is a shell `[ -e path ]` (symlinks followed) with the same third answer as IsFile.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case absent(err):
		return false, nil
	}
	return false, err
}

// IsDir is a shell `[ -d path ]` (symlinks followed) with the same third answer as IsFile.
func IsDir(path string) (bool, error) {
	st, err := os.Stat(path)
	switch {
	case err == nil:
		return st.IsDir(), nil
	case absent(err):
		return false, nil
	}
	return false, err
}

// IsExecutableFile is a shell `[ -f path ] && [ -x path ]`: a regular file this process may execute.
func IsExecutableFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular() && syscall.Access(path, 1) == nil
}

// Reason is the cause of a file error without the operation and path Go puts in front ("permission
// denied", not "open /x: permission denied"), for a message that already names the path.
func Reason(err error) string {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err.Error()
	}
	return err.Error()
}
