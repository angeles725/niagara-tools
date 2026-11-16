package ngdeploy

import (
	"os"
	"path/filepath"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// sweepLockTemps removes the temporary lock files (makeLock) a killed run left in dir: regular files named
// .ng-deploy.lock.<random>.tmp older than lockTempStale, each with a NOTICE. A younger one may be the file
// of a run taking its lock now and is left; one that cannot be removed gets the warning a run gives for its own.
func (d *deploy) sweepLockTemps(dir string) {
	temps, _ := filepath.Glob(filepath.Join(dir, lockName+".*"+lockTempSuffix)) // a pattern error is impossible: the pattern is fixed
	for _, temp := range temps {
		st, err := os.Lstat(temp)
		if err != nil || !st.Mode().IsRegular() || time.Since(st.ModTime()) < lockTempStale {
			continue
		}
		if err := os.Remove(temp); err != nil {
			d.warn(warnLockTempLeft, temp, strerror.Text(err))
			continue
		}
		d.warn(noticeLockTempSwept, temp)
	}
}
