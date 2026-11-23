package buildmodule

import (
	"errors"
	"io"
	"os"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// The chain's file work is done here, not by cp, mkdir, mktemp, unzip and df: a host without one of them
// builds the same, and a test injects a fault with the file system, not with a stub on PATH
// (retro go-build-module-port Δ3).

// tempPattern is the name of a temporary path: mktemp's tmp. and random characters.
const tempPattern = "tmp.*"

// copyPreserving is `cp -p src dst`: the bytes, then the mode, the times and (as far as the user may) the
// owner of src. reason is "" and code 0 when it copied; else why not and 1, the status cp gave. dst is
// written in place (an existing file keeps its inode); one that is the same file as src, or a directory, is
// refused, never truncated.
func (b *build) copyPreserving(src, dst string) (reason string, code int) {
	if err := copyFile(src, dst); err != nil {
		return err.Error(), ExitFault
	}
	return "", 0
}

// copyFile is copyPreserving's work; its error reads "cannot read <src>: <why>" or "cannot write <dst>: <why>".
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fileError("read", src, err)
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return fileError("read", src, err)
	}
	if info.IsDir() {
		return fileError("read", src, syscall.EISDIR) // cp -p omits a directory; nothing is created at dst
	}
	if now, err := os.Stat(dst); err == nil && os.SameFile(info, now) {
		return fileError("write", dst, errors.New("it is the file being copied"))
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fileError("write", dst, err)
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return fileError("write", dst, err)
	}
	if err = out.Close(); err != nil {
		return fileError("write", dst, err)
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		_ = os.Lchown(dst, int(st.Uid), int(st.Gid)) // best effort, as cp -p: a user may not give a file away
	}
	if err = os.Chmod(dst, info.Mode()); err != nil {
		return fileError("write", dst, err)
	}
	if err = os.Chtimes(dst, accessTime(info), info.ModTime()); err != nil {
		return fileError("write", dst, err)
	}
	return nil
}

// accessTime is the access time of info, its modification time where the platform does not say.
func accessTime(info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(int64(st.Atim.Sec), int64(st.Atim.Nsec))
	}
	return info.ModTime()
}

// fileError words a failed file step: "cannot <verb> <path>: <C-locale strerror text>".
func fileError(verb, path string, err error) error {
	return errors.New("cannot " + verb + " " + path + ": " + strerror.Text(err))
}

// makeDir is `mkdir -p dir` (parents too, an existing directory is fine), or `mkdir dir` when parents is
// false (an existing path is an error). code is 1 and a message names why when it fails.
func (b *build) makeDir(dir string, parents bool) (code int) {
	var err error
	if parents {
		err = os.MkdirAll(dir, 0o777)
	} else {
		err = os.Mkdir(dir, 0o777)
	}
	if err != nil {
		b.warn(msgNoDir, dir, strerror.Text(err))
		return ExitFault
	}
	return 0
}
