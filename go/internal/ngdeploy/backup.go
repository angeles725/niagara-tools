package ngdeploy

import (
	"io"
	"os"
	"strconv"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// backup archives what the deploy is about to replace, unless --no-backup (a warning says so). It runs just
// before the copy, after the build and every check, so a run that stops before then leaves _backups as it
// was; with --no-deploy it runs after the build, before the run keeps what gradle installed. The default is
// lightweight: only this module's own rt, ux and wb jars that are deployed. --full-backup or FULL_BACKUP=1
// archives the whole modules directory. After a backup the older ones are purged (purge.go); a purge that
// fails is exit 21 with the new backup in place, before anything is copied.
func (d *deploy) backup() int {
	if d.isOn(optNoBackup) {
		return proceed
	}
	return d.archive()
}

// warnNoBackup prints the warning of --no-backup where the backup step stood before it moved to the copy:
// before anything is built, so a run that stops early still says it will take no backup.
func (d *deploy) warnNoBackup() int {
	if d.isOn(optNoBackup) {
		d.warn(warnBackupSkipped)
	}
	return proceed
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
//
// When the station is <NIAGARA_HOME>/modules the build has replaced its jars by then: the previous ones are the
// files aside (deployedJar), archived under their jar names, and a full backup leaves out what the build put there.
func (d *deploy) archive() int {
	from, members := dirname(d.station), []string{basename(d.station)}
	var options []string
	if d.sameDir && d.isOn(optFullBackup) {
		options = d.sameDirOptions(basename(d.station) + "/")
	} else if d.sameDir {
		options = d.sameDirOptions("")
	}
	if !d.isOn(optFullBackup) {
		from, members = d.station, nil
		for _, profile := range backupProfiles {
			jar := d.deployedJar(d.jarName(profile))
			is, err := regular(jar)
			if err != nil { // not "no such jar": without this the deploy would go on with no backup
				return d.fail(ExitBackup, msgBackupCheck, jar, strerror.Text(err))
			}
			if is {
				members = append(members, basename(jar))
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
	args := append(append([]string{"-czf", name}, options...), "-C", from)
	if d.run(d.s.Out, io.Discard, "tar", append(args, members...)...) != 0 {
		return d.fail(ExitBackup, msgBackupFailed, name)
	}
	if failed, err := d.purgeBackups(name); err != nil {
		return d.fail(ExitPurge, msgPurgeFailed, name, failed, strerror.Text(err))
	}
	d.say(msgBackupOK, name)
	d.backedUp = true
	return proceed
}
