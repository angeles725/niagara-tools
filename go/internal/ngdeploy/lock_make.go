package ngdeploy

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// hardLink gives a lock its name; a test swaps it to take hard links away.
var hardLink = os.Link

// makeLock creates path holding content, complete from the moment path exists, or fails with fs.ErrExist
// when path is there: the content goes into a temporary file beside it, which is then hard-linked to path
// (a link, like O_EXCL, never replaces a file). A run that loses the race reads a whole lock, never an
// empty one. When the link fails for another reason than a lock that is there (a file system without hard
// links, EMLINK and the like), the lock is made in two steps (makeLockInTwoSteps) and inTwoSteps says so:
// the run prints why (noticeLockInTwoSteps) once it holds the lock.
func (d *deploy) makeLock(path, content string) (inTwoSteps error, err error) {
	temp, err := os.CreateTemp(filepath.Dir(path), lockName+".*"+lockTempSuffix)
	if err != nil {
		return nil, makeLockInTwoSteps(path, content) // what O_EXCL says: the lock is there, or the directory may not be written
	}
	err = temp.Chmod(0o644) // the mode of the lock, which is this file
	if err == nil {
		_, err = temp.WriteString(content)
	}
	if closing := temp.Close(); err == nil {
		err = closing
	}
	if err == nil {
		err = hardLink(temp.Name(), path)
	}
	if removing := os.Remove(temp.Name()); removing != nil && !errors.Is(removing, fs.ErrNotExist) {
		d.warn(warnLockTempLeft, temp.Name(), strerror.Text(removing)) // gone is removed: another run swept it
	}
	switch {
	case linkFailed(err) && errors.Is(err, fs.ErrNotExist):
		return nil, makeLockInTwoSteps(path, content) // the temporary file was swept: the file system is not to blame
	case linkFailed(err):
		return err, makeLockInTwoSteps(path, content)
	}
	return nil, err
}

// linkFailed reports whether the link failed for another reason than a lock that is there: a file system
// without hard links (Linux says EPERM on vfat), too many links, and the like.
func linkFailed(err error) bool {
	var link *os.LinkError
	return errors.As(err, &link) && !errors.Is(err, fs.ErrExist)
}

// makeLockInTwoSteps creates path with O_CREATE|O_EXCL and then writes content: a run that starts in the
// same instant can find it empty ("it does not say which run made it"). A lock it could not write is
// removed again: this run made it.
func makeLockInTwoSteps(path, content string) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, err = file.WriteString(content)
	if closing := file.Close(); err == nil {
		err = closing
	}
	if err != nil {
		os.Remove(path)
	}
	return err
}
