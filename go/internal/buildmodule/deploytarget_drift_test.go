package buildmodule

import (
	"strings"
	"testing"
)

// targetMirrorBuild is mirrorBuild (the mirror holds Foo-rt 1.0.0, the build ships other bytes at 1.0.0) with
// --target plant1 recording Foo at recorded.
func (f *fixture) targetMirrorBuild(recorded string) {
	f.t.Helper()
	f.mirrorBuild()
	f.targetRepo("1.0.0", "plant1 Foo@"+recorded+"=%SHA%")
	f.seesBaselines("0")
}

// TestTargetDriftComparesWithTheRecordedVersion: a station that runs the version being shipped is the trap,
// whatever the mirror holds: changed bytes at the recorded version are exit 51 even when the mirror's jar is this
// checkout's own build (which, without a recorded deploy, is skipped).
func TestTargetDriftComparesWithTheRecordedVersion(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.targetMirrorBuild("1.0.0")
	before := f.read(f.deployed("rt"))
	f.write(f.recordPath(), "Foo-rt.jar "+f.digest(f.deployed("rt"))+"\n")
	f.wants("recorded version", f.build("--target", "plant1"), ExitDrift, drift51)
	if strings.Contains(f.err, "own build") || f.read(f.deployed("rt")) != before {
		t.Errorf("skipped the check or did not restore the old jar:\n%s", f.err)
	}
}

// TestTargetDriftSkipsAVersionNoStationRuns: the recorded deploy is another version, so the station does not hold
// the version being shipped and nothing is shipped under a version it knows: no exit 51, one INFO line naming the
// target and both versions, and the new jar stays in modules/. Without --target the same run is exit 51.
func TestTargetDriftSkipsAVersionNoStationRuns(t *testing.T) {
	f := newFixture(t)
	f.signs()
	f.targetMirrorBuild("0.9.0")
	f.wants("other version", f.build("--target", "plant1"), ExitOK,
		"build.sh: INFO — .deploy-targets records Foo@0.9.0 for plant1, not 1.0.0: Foo-rt.jar is not what that station runs, drift check skipped\n")
	if f.read(f.deployed("rt")) != f.read(f.built("rt")) || strings.Contains(f.err, "FAIL") {
		t.Errorf("modules/ does not hold the new jar, or stderr says FAIL:\n%s", f.err)
	}
	f = newFixture(t)
	f.signs()
	f.targetMirrorBuild("0.9.0")
	f.wants("no --target", f.build("--schema-baseline-ref", "v1"), ExitDrift, drift51)
}
