// Package strerror words an OS error as the C library's strerror() does in the C locale, the text GNU
// find and awk print after a path. It is shared by the lint engine and jarversion, which sit side by
// side and must not import each other.
package strerror

import (
	"errors"
	"syscall"
)

// Text is the C-locale strerror text of err's errno (Go's errno strings are glibc's, lower-cased; the
// first letter is upper-cased back), else err's own text unchanged.
func Text(err error) string {
	var errno syscall.Errno
	if !errors.As(err, &errno) {
		return err.Error()
	}
	s := errno.Error()
	if s != "" && s[0] >= 'a' && s[0] <= 'z' {
		s = string(s[0]-'a'+'A') + s[1:]
	}
	return s
}
