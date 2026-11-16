package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// An old backup that cannot be removed stops the deploy before the copy: the new backup is written and
// named, the others are still purged, and nothing was copied.
func TestAPurgeThatCannotRemoveABackupExits21BeforeTheCopy(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	dir := oldBackups(f)
	stuck := dir + "/demo-pre-20260101-000000.tar.gz"
	f.remove(stuck)
	f.write(stuck+"/inside", "a directory in the way of the oldest backup") // os.Remove: Directory not empty
	oldest := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(stuck, oldest, oldest); err != nil {
		t.Fatal(err)
	}
	code := f.run()
	want := "[ng-deploy] ERROR backup written (_backups/demo-pre-TS.tar.gz), but an older backup could not be purged: _backups/demo-pre-20260101-000000.tar.gz: Directory not empty\n"
	written := backupWritten(f.out)
	if code != ExitPurge || strings.Replace(f.err, written, "_backups/demo-pre-TS.tar.gz", 1) != want || !strings.HasSuffix(stamp(f.out), "[ng-deploy] gate ok\n[ng-deploy] backup: _backups/demo-pre-TS.tar.gz\n") {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
	if strings.Count(f.calls(), "tar <-czf>") != 1 || strings.Contains(f.out, "copy ok") {
		t.Errorf("the deploy went on after a failed purge:\n%s", f.calls())
	}
	f.station("old-rt", "old-ux", 0o640)
	kept, _ := filepath.Glob(dir + "/demo-pre-*")
	if len(kept) != 4 || f.exists(dir+"/demo-pre-20260102-000000.tar.gz") || !f.exists(stuck+"/inside") { // the stuck one, the two newest old ones, the new one
		t.Errorf("backups left: %v", kept)
	}
}

// A module name that is not a valid glob pattern cannot list its backups: that is a failed purge too.
func TestAPurgeThatCannotListTheBackupsExits21(t *testing.T) {
	f := newFixture(t)
	f.set("MODULE_NAME", "'de[mo'")
	code := f.run("--full-backup", "--no-deploy") // the built jars of that name are not there for the prechecks of a deploy
	if code != ExitPurge || !strings.Contains(stamp(f.err), "[ng-deploy] ERROR backup written (_backups/de[mo-pre-TS.tar.gz), but an older backup could not be purged: _backups/de[mo-pre-*.tar.gz: syntax error in pattern\n") || strings.Contains(f.out, "no-deploy: ") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

// The backup just written is never purged, whatever the clock of the older ones says: three backups
// stamped later than it (a clock set back, a coarse file system) do not push it out of the kept set.
func TestThePurgeNeverRemovesTheBackupJustWritten(t *testing.T) {
	f := newFixture(t)
	f.deployed()
	dir := oldBackups(f)
	later := time.Now().Add(48 * time.Hour)
	for _, day := range []string{"03", "04", "05"} {
		if err := os.Chtimes(dir+"/demo-pre-202601"+day+"-000000.tar.gz", later, later); err != nil {
			t.Fatal(err)
		}
	}
	code := f.run("--no-deploy")
	written := f.work + "/" + backupWritten(f.out)
	kept, _ := filepath.Glob(dir + "/demo-pre-*")
	if code != ExitOK || !f.exists(written) || len(kept) != 3 || f.exists(dir+"/demo-pre-20260101-000000.tar.gz") {
		t.Errorf("exit %d, written %s, backups left: %v\nstdout %q stderr %q", code, written, kept, f.out, f.err)
	}
}

// backupWritten is the archive a run named on its "backup: " line.
func backupWritten(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if name, ok := strings.CutPrefix(line, "[ng-deploy] backup: "); ok {
			return name
		}
	}
	return ""
}
