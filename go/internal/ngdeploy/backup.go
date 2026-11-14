package ngdeploy

import (
	"io"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// backup archives what the deploy is about to replace, unless --no-backup (a warning says so). The default
// is lightweight: only this module's own rt, ux and wb jars that are deployed. --full-backup or FULL_BACKUP=1
// archives the whole modules directory. After a backup the older ones are purged (purge.go); a purge that
// fails is exit 21 with the new backup in place, before anything is built.
func (d *deploy) backup() int {
	if d.isOn(optNoBackup) {
		d.warn(warnBackupSkipped)
		return proceed
	}
	return d.archive()
}

// archive writes _backups/<MODULE>-pre-<time>.tar.gz in the working directory. A tar that fails is exit 20.
func (d *deploy) archive() int {
	name := backupDir + "/" + d.module + "-pre-" + time.Now().Format(backupStamp) + ".tar.gz"
	if code := d.show("mkdir", "-p", backupDir); code != 0 {
		return code
	}
	d.say(msgBackup, name)
	from, members := dirname(d.station), []string{basename(d.station)}
	if !d.isOn(optFullBackup) {
		from, members = d.station, nil
		for _, profile := range backupProfiles {
			jar := d.station + "/" + d.jarName(profile)
			is, err := regular(jar)
			if err != nil { // not "no such jar": without this the deploy would go on with no backup
				return d.fail(ExitBackup, msgBackupCheck, jar, strerror.Text(err))
			}
			if is {
				members = append(members, d.jarName(profile))
			}
		}
		if len(members) == 0 {
			d.say(msgBackupNothing, d.module, d.station)
			return proceed
		}
	}
	if d.run(d.s.Out, io.Discard, "tar", append([]string{"-czf", name, "-C", from}, members...)...) != 0 {
		return d.fail(ExitBackup, msgBackupFailed, name)
	}
	if failed, err := d.purgeBackups(); err != nil {
		return d.fail(ExitPurge, msgPurgeFailed, name, failed, strerror.Text(err))
	}
	d.say(msgBackupOK, name)
	return proceed
}
