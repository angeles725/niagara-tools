//go:build unix

package license

import (
	"os"
	"syscall"
)

// openRegular opens path for reading only when it is a regular file, or says why not. The open
// does not follow a link put there after the directory was listed, and does not wait on a FIFO;
// what was opened is then checked, so nothing can be swapped in between.
func openRegular(path string) (*os.File, string) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		if info, lerr := os.Lstat(path); lerr == nil && !info.Mode().IsRegular() {
			return nil, SkipNotRegular
		}
		return nil, SkipUnreadable
	}
	if info, err := f.Stat(); err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, SkipNotRegular
	}
	return f, ""
}
