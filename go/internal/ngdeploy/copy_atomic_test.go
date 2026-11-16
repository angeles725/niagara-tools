package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

const (
	copyUnchanged = "nothing was replaced: the station holds the jars it held before the deploy\n"
	copyHalfDone  = "demo-rt.jar in the station is the NEW jar and demo-ux.jar was not replaced: "
)

// upgrade is a fixture whose station holds an older pair, the rt jar with a mode of its own, and whose
// build is a version bump.
func upgrade(t *testing.T, f *fixture) {
	t.Helper()
	f.deployed()
	if err := os.Chmod(f.deployedJar("rt"), 0o640); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
}

// failRename makes the rename onto a path ending in suffix fail, as a station holding the jar open does.
func failRename(t *testing.T, suffix string) {
	t.Helper()
	t.Cleanup(func() { rename = os.Rename })
	rename = func(from, to string) error {
		if strings.HasSuffix(to, suffix) {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EACCES}
		}
		return os.Rename(from, to)
	}
}

// station fails the test unless the two station jars hold these bytes ("" for no jar), the rt jar has this
// mode, and the copy left no staged file behind.
func (f *fixture) station(rt, ux string, rtMode os.FileMode) {
	f.t.Helper()
	if f.read(f.deployedJar("rt")) != rt || f.read(f.deployedJar("ux")) != ux || f.exists(f.deployedJar("rt")) != (rt != "") {
		f.t.Errorf("station holds rt %q ux %q, want %q %q", f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")), rt, ux)
	}
	if st, err := os.Stat(f.deployedJar("rt")); err == nil && st.Mode().Perm() != rtMode {
		f.t.Errorf("demo-rt.jar has mode %o, want %o", st.Mode().Perm(), rtMode)
	}
	if left, _ := filepath.Glob(f.modules + "/*" + stagedSuffix); len(left) != 0 {
		f.t.Errorf("staged files left in the station: %v", left)
	}
}

// Both jars are staged beside their targets and renamed: a stale staged file of a killed run is written
// over, a jar the old copy could not write into (read-only) is replaced, and each jar keeps the mode the
// station's had.
func TestTheCopyStagesBothJarsAndRenamesThem(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	f.chmod(f.deployedJar("ux"), 0o444)
	f.write(f.deployedJar("rt")+stagedSuffix+"/inside", "left by a killed run, and in the way")
	f.write(f.deployedJar("ux")+stagedSuffix, "left by a killed run")
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] gate ok\n[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n")
	f.station("rtjar", "uxjar", 0o640)
	if st, err := os.Stat(f.deployedJar("ux")); err != nil || st.Mode().Perm() != 0o444 || f.err != warnNoBackup {
		t.Errorf("demo-ux.jar: %v %v, stderr %q", st, err, f.err)
	}
}

// A jar that cannot be staged stops the copy before anything is renamed, whichever jar it is.
func TestAJarThatCannotBeStagedLeavesTheStationUnchanged(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	f := newFixture(t)
	upgrade(t, f)
	f.chmod(f.modules, 0o555)
	code := f.run("--no-backup")
	if code != ExitCopy || !strings.HasSuffix(f.err, "[ng-deploy] ERROR copy failed for demo-rt.jar; "+copyUnchanged) || !strings.Contains(f.err, "cp: ") || strings.Count(f.err, "ERROR") != 1 {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	os.Chmod(f.modules, 0o755)
	f.station("old-rt", "old-ux", 0o640)

	f = newFixture(t) // the rt jar is staged, the ux jar is not: the staged rt jar is removed, not renamed
	upgrade(t, f)
	f.chmod(f.builtJar("ux"), 0o000)
	code = f.run("--no-backup", "--no-gate")
	if code != ExitCopy || !strings.HasSuffix(f.err, "[ng-deploy] ERROR copy failed for demo-ux.jar; "+copyUnchanged) || !strings.Contains(f.err, "cp: ") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
	if f.exists(f.work + "/.last-deploy-sha") {
		t.Error("a failed copy recorded the commit")
	}
}

// The first rename fails: the old jar is untouched (a rename replaces a jar whole or not at all).
func TestAFirstRenameThatFailsLeavesTheStationUnchanged(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, "/demo-rt.jar")
	code := f.run("--no-backup")
	if code != ExitCopy || f.err != warnNoBackup+"[ng-deploy] ERROR copy failed for demo-rt.jar: Permission denied; "+copyUnchanged {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
}

// The second rename fails after the first jar went in: the previous first jar is put back from the backup
// this run took, with its bytes and its mode, whether the backup is the module's jars or the whole directory.
func TestASecondRenameThatFailsPutsThePreviousPairBackFromTheBackup(t *testing.T) {
	for _, args := range [][]string{nil, {"--full-backup"}} {
		f := archiveFixture(t)
		upgrade(t, f)
		failRename(t, "/demo-ux.jar")
		code := f.run(args...)
		want := "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; demo-rt.jar was put back from _backups/demo-pre-TS.tar.gz: the station holds the jars it held before the deploy\n"
		if code != ExitCopy || stamp(f.err) != want || strings.Contains(f.out, "copy ok") || strings.Contains(f.out, "restart") {
			t.Errorf("%v: exit %d, stdout:\n%sstderr %q", args, code, f.out, f.err)
		}
		f.station("old-rt", "old-ux", 0o640)
		if f.exists(f.work + "/.last-deploy-sha") {
			t.Errorf("%v: a failed copy recorded the commit", args)
		}
	}
}

// A first jar the station did not have is removed again: that is the pair it held, backup or not.
func TestASecondRenameThatFailsRemovesAJarTheStationDidNotHave(t *testing.T) {
	for _, args := range [][]string{nil, {"--no-backup"}} {
		f := newFixture(t)
		f.write(f.deployedJar("ux"), "uxjar") // the same bytes: no version to compare
		failRename(t, "/demo-ux.jar")
		code := f.run(args...)
		want := "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; demo-rt.jar, which the station did not have, was removed: the station holds the jars it held before the deploy\n"
		if code != ExitCopy || !strings.HasSuffix(f.err, want) {
			t.Errorf("%v: exit %d, stderr %q", args, code, f.err)
		}
		f.station("", "uxjar", 0)
	}
}

// Without a backup there is nothing to put the first jar back from: the line says which jar is new, which
// is not, and what to do before the station restarts.
func TestASecondRenameThatFailsWithoutABackupSaysWhichJarIsNew(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, "/demo-ux.jar")
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyHalfDone +
		"this run took no backup to put the previous demo-rt.jar back from (--no-backup); put it back yourself, or fix the cause and deploy again, before the station restarts\n"
	if code != ExitCopy || f.err != want {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("rtjar", "old-ux", 0o640)
}

// The restore itself fails (the archive cannot be read, or does not hold the jar's bytes): the same two
// facts, the backup to restore from by hand, and the first jar left as the copy put it.
func TestARestoreThatFailsNamesTheBackupToRestoreFrom(t *testing.T) {
	for why, extract := range map[string]string{
		"tar -xzOf exited 2":                            `echo "tar: Unexpected EOF in archive" >&2; exit 2`,
		"the backup does not hold the bytes of the jar": `printf short`,
	} {
		f := newFixture(t)
		upgrade(t, f)
		f.write(f.dir+"/do/tar", `case "$1" in -czf) : > "$2" ;; *) `+extract+` ;; esac`)
		failRename(t, "/demo-ux.jar")
		code := f.run()
		want := "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyHalfDone +
			"putting the previous demo-rt.jar back from _backups/demo-pre-TS.tar.gz failed (" + why + "); restore it from that backup, or fix the cause and deploy again, before the station restarts\n"
		if code != ExitCopy || !strings.HasSuffix(stamp(f.err), want) || strings.Count(f.err, "ERROR") != 1 {
			t.Errorf("%s: exit %d, stderr %q", why, code, f.err)
		}
		f.station("rtjar", "old-ux", 0o640)
		if backups, _ := filepath.Glob(f.work + "/_backups/demo-pre-*.tar.gz"); len(backups) != 1 {
			t.Errorf("%s: backups %v", why, backups)
		}
	}
}
