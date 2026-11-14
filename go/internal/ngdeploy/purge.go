package ngdeploy

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// purgeBackups keeps the newest backups of this module, as many as KEEP_N says, and removes the older
// ones, newest first by modification time and then by name (`ls -t`). Another module's backups are not
// touched. Every older backup is tried; failed names the first one that could not be listed, examined or
// removed (one that is already gone is not a failure), and err says why. The caller stops the deploy.
func (d *deploy) purgeBackups() (failed string, err error) {
	pattern := backupDir + "/" + d.module + "-pre-*.tar.gz"
	names, err := filepath.Glob(pattern)
	if err != nil {
		return pattern, err
	}
	note := func(name string, e error) {
		if err == nil && e != nil && !errors.Is(e, fs.ErrNotExist) {
			failed, err = name, e
		}
	}
	type backup struct {
		name string
		at   int64
	}
	var backups []backup
	for _, name := range names {
		st, e := os.Stat(name)
		if note(name, e); e == nil {
			backups = append(backups, backup{name, st.ModTime().UnixNano()})
		}
	}
	sort.Slice(backups, func(i, j int) bool {
		if backups[i].at != backups[j].at {
			return backups[i].at > backups[j].at
		}
		return backups[i].name < backups[j].name
	})
	for i, b := range backups {
		if i >= d.keep() {
			note(b.name, os.Remove(b.name))
		}
	}
	return failed, err
}
