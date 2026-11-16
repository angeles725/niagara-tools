//go:build !unix

package license

import "os"

// openRegular opens path for reading only when it is a regular file, or says why not. This
// platform has no open that refuses a link, so the file is checked before and after the open.
func openRegular(path string) (*os.File, string) {
	if info, err := os.Lstat(path); err != nil || !info.Mode().IsRegular() {
		if err != nil {
			return nil, SkipUnreadable
		}
		return nil, SkipNotRegular
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, SkipUnreadable
	}
	if info, err := f.Stat(); err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, SkipNotRegular
	}
	return f, ""
}
