package newretro

import (
	"errors"
	"os"
	"syscall"
)

// lock takes an exclusive flock(2) on dir itself (Linux and macOS lock a directory as they lock a file) and
// returns its release. Locking the directory, not a lock file, leaves nothing behind in the tree, and the
// directory is never replaced, so every run locks the same inode (BUILD-STATE.src.md and INDEX.md are
// replaced by rename, so a lock on either could land on a stale inode). The kernel drops a flock when its last
// descriptor closes, process death included, so a killed run never leaves a lock that blocks the next one.
func lock(dir string) (func(), error) {
	f, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	for {
		if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); !errors.Is(err, syscall.EINTR) {
			break
		}
	}
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() { _ = f.Close() }, nil
}
