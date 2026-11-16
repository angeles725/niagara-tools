package ngdeploy

// The run lock (lock.go): one file in each modules directory a run changes, from before the run looks at
// the directory until it has settled it.
const (
	// lockName is the lock file of a modules directory.
	lockName = ".ng-deploy.lock"
	// lockContent takes the pid, the host and the start time (lockTime, UTC) of the run that holds it.
	lockContent = "pid %d\nhost %s\nstarted %s\n"
	lockTime    = "2006-01-02T15:04:05Z"
	// msgLocked takes the lock file, who holds it and how to tell that the run is gone (below).
	msgLocked = "another ng-deploy run holds %s (%s): two runs on one modules directory lose jars. " +
		"A lock is never removed for the run that made it: %s; delete the file and deploy again"
	// holderKnown takes the pid, the host, the start time and the age; goneKnown the pid, the host, the pid.
	holderKnown = "pid %s on host %s, started %s, %s ago"
	goneKnown   = "when no process %s runs on %s (ps -p %s), that run was killed"
	// holderUnknown takes the age of the file.
	holderUnknown = "it does not say which run made it; the file is %s old"
	goneUnknown   = "when no other ng-deploy runs on that directory, that run was killed"
	// lockTempSuffix ends the temporary file a lock is written in before it gets its name (makeLock).
	lockTempSuffix = ".tmp"
	// noticeLockInTwoSteps takes the modules directory and why it took no hard link.
	noticeLockInTwoSteps = "NOTICE: %s takes no hard link (%s): the run lock is made, then written; " +
		"a run that starts in the same instant can find it empty and say that it does not say which run made it"
	// warnLockTempLeft takes the temporary file of a lock and why it could not be removed.
	warnLockTempLeft = "WARN %s, the file this run wrote its lock in, could not be removed (%s): delete it"
	// msgLockFailed takes the lock file and why it could not be made.
	msgLockFailed = "cannot take the run lock %s (%s); nothing was built or changed"
	// warnLockLeft takes the lock file and why this run could not remove its own.
	warnLockLeft = "WARN the run lock %s could not be removed (%s): delete it, the next deploy refuses while it is there"
)
