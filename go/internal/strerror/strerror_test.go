package strerror

import (
	"errors"
	"fmt"
	"io/fs"
	"syscall"
	"testing"
)

func TestText(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		err  error
		want string
	}{
		"EACCES":             {syscall.EACCES, "Permission denied"},
		"ENOENT":             {syscall.ENOENT, "No such file or directory"},
		"ELOOP":              {syscall.ELOOP, "Too many levels of symbolic links"},
		"EIO":                {syscall.EIO, "Input/output error"},
		"unknown errno":      {syscall.Errno(4095), "Errno 4095"},
		"path error":         {&fs.PathError{Op: "open", Path: "/x", Err: syscall.EISDIR}, "Is a directory"},
		"wrapped path error": {fmt.Errorf("read: %w", &fs.PathError{Op: "read", Path: "/x", Err: syscall.EACCES}), "Permission denied"},
		"no errno":           {errors.New("zip: not a valid zip file"), "zip: not a valid zip file"},
		"no errno, empty":    {errors.New(""), ""},
	} {
		if got := Text(tc.err); got != tc.want {
			t.Errorf("%s: Text(%v) = %q, want %q", name, tc.err, got, tc.want)
		}
	}
}
