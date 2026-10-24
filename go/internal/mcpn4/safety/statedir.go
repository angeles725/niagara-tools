package safety

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// StateFiles are the files the server keeps in the state dir.
var StateFiles = []string{"journal.jsonl", "audit.jsonl"}

// Error is safety.SafetyError: a refusal whose text is shown.
type Error struct{ Msg string }

func (e *Error) Error() string { return e.Msg }

// LooseFileReason is loose_file_reason: why an existing journal or audit file is not private,
// "" when it is.
func LooseFileReason(info fs.FileInfo) string {
	st, ok := info.Sys().(*syscall.Stat_t)
	if ok && int(st.Uid) != os.Getuid() {
		return "is owned by another user"
	}
	if mode := info.Mode().Perm(); mode&0o077 != 0 {
		return fmt.Sprintf("is readable or writable by group/others (mode %o)", imode(info))
	}
	return ""
}

// imode is stat.S_IMODE: the permission, set-id and sticky bits as st_mode holds them.
func imode(info fs.FileInfo) uint32 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Mode & 0o7777
	}
	return uint32(info.Mode().Perm())
}

// CheckStateDir is check_state_dir: a missing dir is fine (it is created 0700); an existing one
// must be a directory the current user owns that group and others cannot write, and its state
// files, when present, must be private. Nothing is ever chmodded.
func CheckStateDir(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, syscall.ENOENT) {
		return nil
	}
	if err != nil {
		return &Error{fmt.Sprintf("--state-dir %s cannot be inspected (%s)", path, strerror.Text(err))}
	}
	if !info.IsDir() {
		return &Error{fmt.Sprintf("--state-dir %s exists and is not a directory", path)}
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
		return &Error{fmt.Sprintf("--state-dir %s is owned by another user: choose a directory you "+
			"own (the server never changes its permissions)", path)}
	}
	if info.Mode().Perm()&0o022 != 0 {
		return &Error{fmt.Sprintf("--state-dir %s is group/world-writable (mode %o): run `chmod 700 "+
			"%s` yourself or choose another directory (the server never changes "+
			"the permissions of a directory it did not create)", path, imode(info), path)}
	}
	for _, name := range StateFiles {
		file := Join(path, name)
		fi, err := os.Stat(file)
		if err != nil {
			continue
		}
		if reason := LooseFileReason(fi); reason != "" {
			return &Error{fmt.Sprintf("%s %s: run `chmod 600 %s` yourself or choose another "+
				"--state-dir (the server never changes the permissions of an "+
				"existing file)", file, reason, file)}
		}
	}
	return nil
}
