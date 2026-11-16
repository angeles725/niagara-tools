package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// Before the build, each station jar the run will replace is renamed to <jar>.ng-probe and straight back: a
// rename the file system refuses (a running station holds the jar) stops the run with exit 40, the line that
// names a running station, and no gradle call (retro 2026-11-15-coordination-stretch-deltas D2).
func TestAStationJarThatCannotBeMovedIsRefusedBeforeTheBuild(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, "/demo-ux.jar"+probeSuffix)
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR copy refused: demo-ux.jar in " + f.modules + " cannot be moved (Permission denied); nothing was built or copied\n" + heldOpenLine
	if code != ExitCopy || !strings.HasSuffix(f.err, want) || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
	f.station("old-rt", "old-ux", 0o640)
	if left, _ := filepath.Glob(f.modules + "/*" + probeSuffix); len(left) != 0 {
		t.Errorf("probe files left: %v", left)
	}
}

// A jar the probe moved and could not move back is named with both paths and the manual fix: exit 40, no
// gradle call, and the jar's bytes are in the probe file.
func TestAProbeThatCannotMoveTheJarBackNamesBothPaths(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, "/demo-ux.jar"+probeSuffix+">/demo-ux.jar")
	code := f.run("--no-backup")
	jar, probe := f.deployedJar("ux"), f.deployedJar("ux")+probeSuffix
	want := "[ng-deploy] ERROR copy refused: " + jar + " was moved to " + probe + " to test it and could not be moved back (Permission denied): " +
		"rename " + probe + " back to " + jar + " before the station restarts; nothing was built or copied\n"
	if code != ExitCopy || !strings.HasSuffix(f.err, want) || strings.Contains(f.calls(), "gradlew") || f.read(probe) != "old-ux" {
		t.Errorf("exit %d, stderr %q, calls %q", code, f.err, f.calls())
	}
}

// A probe file a killed run left beside the jar is never written over: the run cannot tell which one the
// station should load, so it refuses with exit 40 and the manual fix, before the build, and changes nothing
// (review R4-stale-probe-blocks-deploys).
func TestAProbeFileBesideTheJarIsRefusedWithTheFix(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	jar := f.deployedJar("rt")
	f.write(jar+probeSuffix, "the only copy")
	code := f.run("--no-backup")
	want := "[ng-deploy] ERROR copy refused: " + jar + probeSuffix + ", left by a run that was killed while it tested the jar, is there beside demo-rt.jar; nothing was built or copied\n" +
		"[ng-deploy]   compare " + jar + " and " + jar + probeSuffix + ": when they hold the same bytes delete " + jar + probeSuffix +
		"; otherwise keep the one the station should load as " + jar + " and delete the other; then deploy again\n"
	if code != ExitCopy || !strings.HasSuffix(f.err, want) || f.read(jar+probeSuffix) != "the only copy" || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
}

// A probe file a killed run left with no jar beside it is the jar: it is moved back under its name with a
// NOTICE, and the deploy goes on (it does not block every later deploy).
func TestAProbeFileWithoutItsJarIsMovedBackAndTheDeployGoesOn(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	jar := f.deployedJar("rt")
	if err := os.Rename(jar, jar+probeSuffix); err != nil {
		t.Fatal(err)
	}
	f.want(f.run("--no-backup"), ExitOK, "[ng-deploy] version-bump ok (demo-rt.jar: 1.0 -> 1.1)\n", "[ng-deploy] copy ok\n",
		"err:[ng-deploy] NOTICE: "+jar+probeSuffix+", left by a run that was killed while it tested the jar, was moved back to "+jar+"\n")
	f.station("rtjar", "uxjar", 0o640)
	if f.exists(jar + probeSuffix) {
		t.Error("the probe file is still there")
	}
}

// A SIGINT, SIGTERM or SIGHUP between the two moves of the probe waits until the jar is back under its name,
// then ends the run with 128 + the signal before the build: no probe file is left (review R1/R3/R4 probe
// signal window).
func TestASignalDuringTheProbeLeavesTheJarInPlace(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP} {
		f := newFixture(t)
		upgrade(t, f)
		signalDuring(t, sig, "/demo-ux.jar"+probeSuffix)
		code := f.run("--no-backup")
		want := "[ng-deploy] ERROR interrupted (" + signalNames[sig] + ") before the copy; nothing was copied to the station\n"
		if code != signalBase+int(sig) || !strings.HasSuffix(f.err, want) || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%v: exit %d, stderr %q", sig, code, f.err)
		}
		f.station("old-rt", "old-ux", 0o640)
		if left, _ := filepath.Glob(f.modules + "/*" + probeSuffix); len(left) != 0 {
			t.Errorf("%v: probe files left: %v", sig, left)
		}
	}
}

// --no-deploy replaces nothing in the station: nothing is probed there, even a jar that cannot be moved.
func TestNoDeployProbesNothing(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	failRename(t, probeSuffix)
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] no-deploy: jars in build/libs/\n")
}

// A station directory that takes no rename refuses at the probe, before the build: exit 40 as before, now
// with no gradle call.
func TestAReadOnlyStationIsRefusedAtTheProbe(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	f := newFixture(t)
	upgrade(t, f)
	f.chmod(f.modules, 0o555)
	code := f.run("--no-backup")
	if code != ExitCopy || !strings.Contains(f.err, "copy refused: demo-rt.jar in "+f.modules+" cannot be moved (Permission denied)") || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}
