package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// lock takes the run lock of a modules directory: a file one run at most can make (makeLock: no flock, so
// a Windows-backed mount keeps the rule), holding who made it, program included, from the moment it is
// there. A lock that is there is another run's, alive or killed: the deploy stops with exit 10 and a line
// that says whose it is, how old, and how to tell that the run is gone (a second line when its pid now runs
// another program); it is never removed for that run (the doctrine of the jars left aside: nothing is
// deleted that this run did not make). A directory the run is not allowed to write
// gets no lock: the run cannot change it either. From its first lock on, the run catches SIGINT and
// SIGTERM (interrupted), so that it ends through unlock.
func (d *deploy) lock(dir string) int {
	path := dir + "/" + lockName
	host, _ := os.Hostname()
	err := d.makeLock(path, fmt.Sprintf(lockContent, os.Getpid(), host, time.Now().UTC().Format(lockTime), ownCommand()))
	switch {
	case errors.Is(err, fs.ErrExist):
		holder, gone, hint := lockedBy(path)
		d.fail(ExitEnv, msgLocked, path, holder, gone)
		if hint != "" {
			d.warn(indent + hint)
		}
		return ExitEnv
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.EROFS):
		return proceed // a directory this run may not write: it cannot change it either, and the step that tries says so
	case err != nil:
		return d.fail(ExitEnv, msgLockFailed, path, strerror.Text(err))
	}
	d.locks = append(d.locks, path)
	if d.signals == nil {
		d.signals = watchInterrupts()
	}
	return proceed
}

// age is the time since a moment, to the second.
func age(since time.Time) string { return time.Since(since).Round(time.Second).String() }

// unlock removes the locks this run took, last taken first, on every way out of Main.
func (d *deploy) unlock() {
	for i := len(d.locks) - 1; i >= 0; i-- {
		if err := os.Remove(d.locks[i]); err != nil {
			d.warn(warnLockLeft, d.locks[i], strerror.Text(err))
		}
	}
	d.locks = nil
}
