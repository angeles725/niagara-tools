package ngdeploy

import (
	"os"
	"path/filepath"
	"sort"
)

// purgeBackups keeps the newest backups of this module, as many as KEEP_N says, and removes the older
// ones, newest first by modification time and then by name (`ls -t`). Another module's backups are not
// touched. As in the bash tool the purge is best effort: a backup it cannot list or remove stays.
func (d *deploy) purgeBackups() {
	names, _ := filepath.Glob(backupDir + "/" + d.module + "-pre-*.tar.gz")
	type backup struct {
		name string
		at   int64
	}
	var backups []backup
	for _, name := range names {
		if st, err := os.Stat(name); err == nil {
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
			_ = os.Remove(b.name)
		}
	}
}
