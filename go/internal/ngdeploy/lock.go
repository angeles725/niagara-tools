package ngdeploy

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"syscall"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// lockHolder reads the content a run writes into its lock (lockContent).
var lockHolder = regexp.MustCompile(`\Apid (\d+)\nhost (\S+)\nstarted (\S+)\n\z`)

// lock takes the run lock of a modules directory: a file made with O_CREATE|O_EXCL, which one run at most
// can make (no flock: a Windows-backed mount keeps the rule), holding who made it. A lock that is there is
// another run's, alive or killed: the deploy stops with exit 10 and a line that says whose it is, how old,
// and how to tell that the run is gone; it is never removed for that run (the doctrine of the jars left
// aside: nothing is deleted that this run did not make). A directory the run is not allowed to write gets no lock:
// the run cannot change it either. From its first lock on, the run catches SIGINT
// and SIGTERM (interrupted), so that it ends through unlock.
func (d *deploy) lock(dir string) int {
	path := dir + "/" + lockName
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	switch {
	case errors.Is(err, fs.ErrExist):
		holder, gone := lockedBy(path)
		return d.fail(ExitEnv, msgLocked, path, holder, gone)
	case errors.Is(err, fs.ErrPermission), errors.Is(err, syscall.EROFS):
		return proceed // a directory this run may not write: it cannot change it either, and the step that tries says so
	case err != nil:
		return d.fail(ExitEnv, msgLockFailed, path, strerror.Text(err))
	}
	d.locks = append(d.locks, path)
	if d.signals == nil {
		d.signals = watchInterrupts()
	}
	host, _ := os.Hostname()
	_, err = fmt.Fprintf(file, lockContent, os.Getpid(), host, time.Now().UTC().Format(lockTime))
	if closing := file.Close(); err == nil {
		err = closing
	}
	if err != nil {
		return d.fail(ExitEnv, msgLockFailed, path, strerror.Text(err))
	}
	return proceed
}

// lockedBy says who holds a lock and how to tell that its run is gone: from its content, or, when that is
// not what a run writes, from the age of the file alone.
func lockedBy(path string) (holder, gone string) {
	content, _ := os.ReadFile(path)
	if m := lockHolder.FindStringSubmatch(string(content)); m != nil {
		if started, err := time.Parse(lockTime, m[3]); err == nil {
			return fmt.Sprintf(holderKnown, m[1], m[2], m[3], age(started)), fmt.Sprintf(goneKnown, m[1], m[2], m[1])
		}
	}
	made := time.Now()
	if st, err := os.Lstat(path); err == nil {
		made = st.ModTime()
	}
	return fmt.Sprintf(holderUnknown, age(made)), goneUnknown
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
