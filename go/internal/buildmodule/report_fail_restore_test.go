package buildmodule

import (
	"strings"
	"testing"
)

// TestReportFailRestoresTheMirrorJar pins P4: a build whose report-module punch-list FAILs (exit 50) puts the
// pre-build jar back into the mirror's modules/, as a failing verify gate does, and keeps nothing under
// .mirror-replaced/; so a fixed build at the same version passes the drift gate without --no-drift-check.
func TestReportFailRestoresTheMirrorJar(t *testing.T) {
	f, oldSum := mirrorBuild(t)
	before := f.read(f.deployed("rt"))
	f.does(reportScript, "exit 1")
	f.wants("report FAIL", f.run(f.root, "Foo", f.home), ExitGate, msgReportFails,
		"  Restored the previously-deployed Foo-rt.jar into "+f.home+"/modules.")
	if f.read(f.deployed("rt")) != before {
		t.Fatalf("the mirror still holds the jar that failed the report")
	}
	if exists(f.home+"/.mirror-replaced/Foo-rt.jar."+oldSum) || strings.Contains(f.out, "==> mirror:") {
		t.Errorf("a FAILed build kept a replaced jar: %q", f.out)
	}

	f.jar(f.built("rt"), "1.0.1", "FIXED payload")
	f.does(reportScript, "exit 0")
	f.wants("fix at the same version", f.run(f.root, "Foo", f.home), ExitOK)
}

// TestReportIncompleteKeepsTheNewJar: only a FAIL rolls the mirror back; an INCOMPLETE (exit 53) or an
// environment error (exit 10) report leaves the new jar, as before.
func TestReportIncompleteKeepsTheNewJar(t *testing.T) {
	for code, want := range map[string]int{"4": ExitReportPartial, "3": ExitEnv} {
		f, _ := mirrorBuild(t)
		f.does(reportScript, "exit "+code)
		f.wants("report exit "+code, f.run(f.root, "Foo", f.home), want)
		if f.read(f.deployed("rt")) != f.read(f.built("rt")) {
			t.Errorf("report exit %s: the new jar was taken out of modules/", code)
		}
	}
}

// firstDeployBuild is a mirror fixture with no Foo-rt in modules/ before the build, whose gradle installs one.
func firstDeployBuild(t *testing.T) *fixture {
	f := newFixture(t)
	f.signs()
	f.write(f.home+"/"+mirrorMarker, "")
	f.jar(f.built("rt"), "1.0.1", "NEW payload")
	f.does("gradlew", `cp "`+f.built("rt")+`" "`+f.deployed("rt")+`"`)
	return f
}

// TestReportFailOnAFirstDeployRemovesTheNewJar (verifier probe, P4 first deploy): with no jar of the profile in
// the mirror before the build, a report FAIL takes the newly-installed jar out of modules/.
func TestReportFailOnAFirstDeployRemovesTheNewJar(t *testing.T) {
	f := firstDeployBuild(t)
	f.does(reportScript, "exit 1")
	f.wants("first-deploy report FAIL", f.run(f.root, "Foo", f.home), ExitGate, msgReportFails,
		"  Removed the newly-installed Foo-rt.jar from "+f.home+"/modules (none was deployed before this build).")
	if exists(f.deployed("rt")) {
		t.Errorf("the jar that failed the report is left in modules/")
	}
}

// TestUnexpectedReportExitRollsBackAsAFail (verifier probe) pins the CURRENT behaviour: any report-module status
// other than 0, 3 and 4 (here 2 and 127, a missing script) reads as a FAIL, so the chain exits 50 and rolls the
// mirror jar back like one.
func TestUnexpectedReportExitRollsBackAsAFail(t *testing.T) {
	for _, code := range []string{"2", "127"} {
		f, _ := mirrorBuild(t)
		before := f.read(f.deployed("rt"))
		f.does(reportScript, "exit "+code)
		f.wants("report exit "+code, f.run(f.root, "Foo", f.home), ExitGate, msgReportFails,
			"  Restored the previously-deployed Foo-rt.jar into "+f.home+"/modules.")
		if f.read(f.deployed("rt")) != before {
			t.Errorf("report exit %s: the mirror was not rolled back", code)
		}
	}
}
