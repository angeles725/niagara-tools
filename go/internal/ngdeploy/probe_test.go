package ngdeploy

import (
	"os"
	"path/filepath"
	"strings"
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

// A probe file left by a killed run is never written over: the run refuses with exit 40 and leaves it.
func TestAProbeNeverReplacesAFileAlreadyThere(t *testing.T) {
	f := newFixture(t)
	upgrade(t, f)
	f.write(f.deployedJar("rt")+probeSuffix, "the only copy")
	code := f.run("--no-backup")
	if code != ExitCopy || !strings.Contains(f.err, "copy refused: demo-rt.jar in "+f.modules+" cannot be moved (File exists)") || f.read(f.deployedJar("rt")+probeSuffix) != "the only copy" || strings.Contains(f.calls(), "gradlew") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f.station("old-rt", "old-ux", 0o640)
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
