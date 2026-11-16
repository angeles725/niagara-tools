package ngdeploy

import (
	"strings"
	"testing"
)

// The backup and its purge run just before the copy, after the build and every check: a run that fails at
// the build or at the gate leaves _backups byte for byte as it was, older backups included, and a run with
// no _backups makes none (retro 2026-11-15-coordination-stretch-deltas D4).
func TestARunThatFailsBeforeTheCopyLeavesTheBackupsAsTheyWere(t *testing.T) {
	for name, fail := range map[string]string{"the build": "FAKE_GRADLEW_EXIT", "the gate": "FAKE_VM_EXIT"} {
		for _, seeded := range []bool{true, false} {
			f := newFixture(t)
			upgrade(t, f)
			f.set("KEEP_N", "1")
			before := ""
			if seeded {
				oldBackups(f)
				before = tree(t, f.work+"/"+backupDir)
			}
			t.Setenv(fail, "1")
			code := f.run()
			if code == ExitOK || strings.Contains(f.calls(), "tar") || strings.Contains(f.out, "backup") {
				t.Errorf("%s, seeded %v: exit %d, stdout:\n%s", name, seeded, code, f.out)
			}
			if seeded && tree(t, f.work+"/"+backupDir) != before {
				t.Errorf("%s: _backups changed:\n%s", name, tree(t, f.work+"/"+backupDir))
			}
			if !seeded && f.exists(f.work+"/"+backupDir) {
				t.Errorf("%s: a run that failed made _backups", name)
			}
		}
	}
}

// A deploy that goes through writes the backup after the gate and before the copy, as one line pair.
func TestTheBackupIsWrittenAfterTheChecksAndBeforeTheCopy(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	f.want(f.run(), ExitOK)
	if !strings.Contains(stamp(f.out), "[ng-deploy] gate ok\n[ng-deploy] backup: _backups/demo-pre-TS.tar.gz\n[ng-deploy] backup ok: _backups/demo-pre-TS.tar.gz\n[ng-deploy] copy ok\n") {
		t.Errorf("stdout:\n%s", f.out)
	}
}
