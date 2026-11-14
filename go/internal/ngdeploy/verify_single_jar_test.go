package ngdeploy

import (
	"strings"
	"testing"
)

// These tests replace the tests/ng-deploy.bats cases that sourced the launcher and called verify_jar or
// verify_cachebuster on their own. Each runs a whole deploy of one jar (mode C for the rt jar, mode B for the
// ux jar), so the step is reached through Main and its effect on the deploy is asserted too.

func TestVerifyAcceptsAJarWithTheExpectedTypeCount(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "9")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] verify ok (demo-rt.jar: 9/9)\n", "[ng-deploy] copy ok\n")
	if f.err != warnNoBackup || f.read(f.deployedJar("rt")) != "rtjar" {
		t.Errorf("stderr %q, deployed rt jar %q", f.err, f.read(f.deployedJar("rt")))
	}
}

func TestVerifyRefusesAJarWithAnotherTypeCount(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "8")
	code := f.run("--no-backup", "--mode", "C")
	want := warnNoBackup + "[ng-deploy] ERROR verify failed: expected 9 <type entries in " + f.builtJar("rt") + ", found 8\n"
	if code != ExitVerify || f.err != want || strings.Contains(f.out, "verify ok") || f.exists(f.deployedJar("rt")) {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
}

// A real module.xml wraps its <type .../> entries in a <types> element: the wrapper and its closing tag are
// not entries (lesson B8: counting "<type" without the space found one more than the jar declares).
func TestVerifyCountsTheTypeEntriesOfARealModuleXML(t *testing.T) {
	f := newFixture(t)
	f.write(f.dir+"/do/unzip", "echo '<types>'\nfor i in 1 2 3; do printf '<type name=\"FakeType%s\"/>\\n' \"$i\"; done\necho '</types>'\n")
	f.set("EXPECTED_RT_TYPES", "3")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] verify ok (demo-rt.jar: 3/3)\n", "[ng-deploy] copy ok\n")
	if f.err != warnNoBackup || !f.exists(f.deployedJar("rt")) {
		t.Errorf("stderr %q", f.err)
	}
}

func TestTheCacheBusterIsNotCheckedWithoutABuildID(t *testing.T) {
	f := newFixture(t) // no BUILD_ID in the environment or the env file, and an index.html with no ?v= at all
	f.want(f.run("--no-backup", "--mode", "B"), ExitOK, "[ng-deploy] verify ok (demo-ux.jar: 9/9)\n", "[ng-deploy] copy ok\n")
	if f.err != warnNoBackup || strings.Contains(f.out, "cache-buster") || strings.Contains(f.calls(), indexHTML) || !f.exists(f.deployedJar("ux")) {
		t.Errorf("stdout %q, stderr %q, calls:\n%s", f.out, f.err, f.calls())
	}
}

func TestTheCacheBusterOfAnotherBuildIsRefused(t *testing.T) {
	f := newFixture(t)
	t.Setenv("BUILD_ID", "abc123") // from the environment, as an operator exports it
	t.Setenv("FAKE_INDEX", "?v=WRONG")
	code := f.run("--no-backup", "--mode", "B")
	want := warnNoBackup + "[ng-deploy] ERROR verify failed: index.html in " + f.builtJar("ux") + " does not contain ?v=abc123 (BUILD_ID check)\n"
	if code != ExitVerify || f.err != want || strings.Contains(f.out, "cache-buster") || f.exists(f.deployedJar("ux")) {
		t.Errorf("exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
	if !strings.Contains(f.calls(), "unzip <-p> <"+f.builtJar("ux")+"> <"+indexHTML+">\n") {
		t.Errorf("calls:\n%s", f.calls())
	}
}
