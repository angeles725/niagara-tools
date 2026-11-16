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
	copyUndone    = "what was already replaced was put back: the station holds the jars it held before the deploy\n"
	copyMixed     = "the station does NOT hold the jars it held before the deploy: "
	copyMixedDo   = "; put each named .ng-deploy-old file back under its jar name before the station restarts\n"
	renameInUX    = "/demo-ux.jar.ng-deploy-new>/demo-ux.jar"
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

// failRename makes some renames fail, as a station holding a jar open does. Each of which is
// "<end of the old path>><end of the new path>", or the end of the new path alone.
func failRename(t *testing.T, which ...string) {
	t.Helper()
	t.Cleanup(func() { rename, link = os.Rename, os.Link })
	failing := func(really func(string, string) error) func(string, string) error {
		return func(from, to string) error {
			for _, w := range which {
				old, new, both := strings.Cut(w, ">")
				if !both {
					old, new = "", w
				}
				if strings.HasSuffix(from, old) && strings.HasSuffix(to, new) {
					return &os.LinkError{Op: "rename", Old: from, New: to, Err: syscall.EACCES}
				}
			}
			return really(from, to)
		}
	}
	rename, link = failing(os.Rename), failing(os.Link) // a jar is moved aside with link, or with rename when link fails
}

// station fails the test unless the two station jars hold these bytes ("" for no jar), the rt jar has this
// mode, and the copy left no staged jar and no jar moved aside behind.
func (f *fixture) station(rt, ux string, rtMode os.FileMode) {
	f.t.Helper()
	if f.read(f.deployedJar("rt")) != rt || f.read(f.deployedJar("ux")) != ux || f.exists(f.deployedJar("rt")) != (rt != "") {
		f.t.Errorf("station holds rt %q ux %q, want %q %q", f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")), rt, ux)
	}
	if st, err := os.Stat(f.deployedJar("rt")); err == nil && st.Mode().Perm() != rtMode {
		f.t.Errorf("demo-rt.jar has mode %o, want %o", st.Mode().Perm(), rtMode)
	}
	if left, _ := filepath.Glob(f.modules + "/*.ng-deploy-*"); len(left) != 0 {
		f.t.Errorf("files of the copy left in the station: %v", left)
	}
}

// Both jars are staged beside their targets, the station's are moved aside, and the staged ones renamed
// in: a stale staged file of a killed run is written over, a read-only jar is replaced, each jar keeps the
// mode the station's had, and nothing of the copy is left, with a backup or without.
func TestTheCopyStagesBothJarsAndRenamesThem(t *testing.T) {
	for _, args := range [][]string{{"--no-backup"}, nil} {
		f := newFixture(t)
		upgrade(t, f)
		f.chmod(f.deployedJar("ux"), 0o444)
		f.write(f.deployedJar("rt")+stagedSuffix+"/inside", "left by a killed run, and in the way")
		f.write(f.deployedJar("ux")+stagedSuffix, "left by a killed run")
		f.want(f.run(args...), ExitOK, "[ng-deploy] gate ok\n", "[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n")
		f.station("rtjar", "uxjar", 0o640)
		if st, err := os.Stat(f.deployedJar("ux")); err != nil || st.Mode().Perm() != 0o444 || strings.Contains(f.err, "ERROR") {
			t.Errorf("%v: demo-ux.jar: %v %v, stderr %q", args, st, err, f.err)
		}
	}
}

// A jar that cannot be staged stops the copy before anything is moved, whichever jar it is.
func TestAJarThatCannotBeStagedLeavesTheStationUnchanged(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	f := newFixture(t)
	upgrade(t, f)
	f.chmod(f.builtJar("rt"), 0o644) // restored at the end of the test
	// unreadable once the gate runs: after the probe before the build, before the copy
	f.write(f.dir+"/do/vm.sh", "command -p chmod 000 "+f.builtJar("rt")+"\n"+stubDo["vm.sh"])
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

// A jar of the station that cannot be moved aside (the station holds it open) leaves everything as it
// was: a jar already moved aside is moved back, and no new jar went in.
func TestAJarThatCannotBeMovedAsideLeavesTheStationUnchanged(t *testing.T) {
	for _, jar := range []string{"demo-rt.jar", "demo-ux.jar"} {
		f := newFixture(t)
		upgrade(t, f)
		failRename(t, "/"+jar+asideSuffix)
		code := f.run("--no-backup")
		if code != ExitCopy || f.err != warnNoBackup+"[ng-deploy] ERROR copy failed for "+jar+": Permission denied; "+copyUnchanged+heldOpenLine {
			t.Errorf("%s: exit %d, stderr %q", jar, code, f.err)
		}
		f.station("old-rt", "old-ux", 0o640)
	}
}

// The first rename in fails: the jars moved aside are moved back.
func TestAFirstRenameThatFailsLeavesTheStationUnchanged(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, "/demo-rt.jar.ng-deploy-new>/demo-rt.jar")
	code := f.run("--no-backup")
	if code != ExitCopy || f.err != warnNoBackup+"[ng-deploy] ERROR copy failed for demo-rt.jar: Permission denied; "+copyUnchanged+heldOpenLine {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
}

// The second rename in fails after the first jar went in: the previous pair is moved back, bytes and mode,
// from the files moved aside, whether or not the run took a backup.
func TestASecondRenameThatFailsPutsThePreviousPairBack(t *testing.T) {
	for _, args := range [][]string{{"--no-backup"}, nil, {"--full-backup"}} {
		f := newFixture(t)
		upgrade(t, f)
		failRename(t, renameInUX)
		code := f.run(args...)
		want := "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyUndone + heldOpenLine
		if code != ExitCopy || !strings.HasSuffix(f.err, want) || strings.Count(f.err, "ERROR") != 1 || strings.Contains(f.out, "copy ok") || strings.Contains(f.out, "restart") {
			t.Errorf("%v: exit %d, stdout:\n%sstderr %q", args, code, f.out, f.err)
		}
		f.station("old-rt", "old-ux", 0o640)
		if f.exists(f.work+"/.last-deploy-sha") || strings.Contains(f.calls(), "tar <-x") {
			t.Errorf("%v: a failed copy recorded the commit, or read the backup:\n%s", args, f.calls())
		}
	}
}

// A first jar the station did not have is removed again: that is the pair it held.
func TestASecondRenameThatFailsRemovesAJarTheStationDidNotHave(t *testing.T) {
	f := newFixture(t)
	f.write(f.deployedJar("ux"), "uxjar") // the same bytes: no version to compare
	failRename(t, renameInUX)
	code := f.run("--no-backup")
	if code != ExitCopy || f.err != warnNoBackup+"[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; "+copyUndone+heldOpenLine {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("", "uxjar", 0)
}

// The only way to a mixed pair: a jar moved aside cannot be moved back. The line says which jar is new,
// which is the previous one, and names the file that still holds the previous bytes; that file stays.
func TestARenameBackThatFailsNamesTheFileThatHoldsThePreviousJar(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, renameInUX, "/demo-rt.jar.ng-deploy-old>/demo-rt.jar")
	code := f.run("--no-backup")
	want := warnNoBackup + "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyMixed +
		"demo-rt.jar is the NEW jar and the previous one is in demo-rt.jar.ng-deploy-old (Permission denied), demo-ux.jar is the previous jar" + copyMixedDo + heldOpenLine
	if code != ExitCopy || f.err != want {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	left, _ := filepath.Glob(f.modules + "/*.ng-deploy-*")
	if f.read(f.deployedJar("rt")) != "rtjar" || f.read(f.deployedJar("ux")) != "old-ux" || f.read(f.deployedJar("rt")+asideSuffix) != "old-rt" || len(left) != 1 {
		t.Errorf("station: rt %q ux %q, files of the copy %v", f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")), left)
	}

	f = newFixture(t) // the jar that failed cannot be moved back either: it is missing, and its bytes are named too
	upgrade(t, f)
	failRename(t, renameInUX, "/demo-ux.jar.ng-deploy-old>/demo-ux.jar")
	code = f.run("--no-backup")
	want = warnNoBackup + "[ng-deploy] ERROR copy failed for demo-ux.jar: Permission denied; " + copyMixed +
		"demo-rt.jar is the previous jar, demo-ux.jar is MISSING and the previous one is in demo-ux.jar.ng-deploy-old (Permission denied)" + copyMixedDo + heldOpenLine
	if code != ExitCopy || f.err != want || f.read(f.deployedJar("rt")) != "old-rt" || f.exists(f.deployedJar("ux")) || f.read(f.deployedJar("ux")+asideSuffix) != "old-ux" {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}
