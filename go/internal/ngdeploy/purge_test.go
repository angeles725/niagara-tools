package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// An old backup that cannot be removed stops the deploy before the build: the new backup is written and
// named, the others are still purged, and nothing else ran.
func TestAPurgeThatCannotRemoveABackupExits21BeforeTheBuild(t *testing.T) {
	f := newFixture(t)
	f.deployed()
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
	written := strings.TrimSuffix(strings.TrimPrefix(f.out, "[ng-deploy] backup: "), "\n")
	if code != ExitPurge || strings.Replace(f.err, written, "_backups/demo-pre-TS.tar.gz", 1) != want || stamp(f.out) != "[ng-deploy] backup: _backups/demo-pre-TS.tar.gz\n" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
	if strings.Contains(f.calls(), "gradlew") || strings.Count(f.calls(), "tar <-czf>") != 1 || f.read(f.deployedJar("rt")) != "old-rt" {
		t.Errorf("the deploy went on after a failed purge:\n%s", f.calls())
	}
	kept, _ := filepath.Glob(dir + "/demo-pre-*")
	if len(kept) != 4 || f.exists(dir+"/demo-pre-20260102-000000.tar.gz") || !f.exists(stuck+"/inside") { // the stuck one, the two newest old ones, the new one
		t.Errorf("backups left: %v", kept)
	}
}

// A module name that is not a valid glob pattern cannot list its backups: that is a failed purge too.
func TestAPurgeThatCannotListTheBackupsExits21(t *testing.T) {
	f := newFixture(t)
	f.set("MODULE_NAME", "'de[mo'")
	code := f.run("--full-backup")
	if code != ExitPurge || !strings.Contains(stamp(f.err), "[ng-deploy] ERROR backup written (_backups/de[mo-pre-TS.tar.gz), but an older backup could not be purged: _backups/de[mo-pre-*.tar.gz: syntax error in pattern\n") || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}
