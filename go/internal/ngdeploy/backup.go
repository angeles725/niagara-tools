package ngdeploy

import (
	"io"
	"os"
	"strconv"
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

// clock is the time a backup is named after; a test replaces it.
var clock = time.Now

// backupName is _backups/<MODULE>-pre-<time>.tar.gz, or with -2, -3... before the suffix when a backup of
// the same second is already there: tar would write over it.
func (d *deploy) backupName() string {
	base := backupDir + "/" + d.module + "-pre-" + clock().Format(backupStamp)
	name := base + backupSuffix
	for n := 2; ; n++ {
		if _, err := os.Lstat(name); err != nil {
			return name
		}
		name = base + "-" + strconv.Itoa(n) + backupSuffix
	}
}

// archive writes the backup in the working directory, under backupDir. A tar that fails is exit 20. The
// archive is named, and backupDir made, only once there is something to archive: a module with no jar in the
// station gets one line that says so.
func (d *deploy) archive() int {
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
	name := d.backupName()
	if code := d.show("mkdir", "-p", backupDir); code != 0 {
		return code
	}
	d.say(msgBackup, name)
	if d.run(d.s.Out, io.Discard, "tar", append([]string{"-czf", name, "-C", from}, members...)...) != 0 {
		return d.fail(ExitBackup, msgBackupFailed, name)
	}
	if failed, err := d.purgeBackups(name); err != nil {
		return d.fail(ExitPurge, msgPurgeFailed, name, failed, strerror.Text(err))
	}
	d.say(msgBackupOK, name)
	d.backedUp = true
	return proceed
}
