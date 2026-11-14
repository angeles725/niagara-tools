package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

func TestADeployCopiesTheJarsOfTheModeAfterEveryCheck(t *testing.T) {
	for _, tc := range []struct {
		mode   string
		rt, ux bool
		last   string
	}{
		{"A", true, true, "[ng-deploy] gate ok\n[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n"},
		{"B", false, true, "[ng-deploy] verify ok (demo-ux.jar: 9/9)\n[ng-deploy] copy ok\n[ng-deploy] browser hard-reload only (no station restart needed)\n"},
		{"C", true, false, "[ng-deploy] gate ok\n[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n"},
	} {
		f := newFixture(t)
		code := f.run("--no-backup", "--mode", tc.mode)
		if code != ExitOK || !strings.HasSuffix(f.out, tc.last) || f.err != warnNoBackup {
			t.Errorf("mode %s: exit %d, stdout:\n%sstderr %q", tc.mode, code, f.out, f.err)
		}
		if (f.read(f.deployedJar("rt")) == "rtjar") != tc.rt || (f.read(f.deployedJar("ux")) == "uxjar") != tc.ux {
			t.Errorf("mode %s: station holds rt %q ux %q", tc.mode, f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")))
		}
	}
}

func TestAFailedCopyExits40NamingTheJar(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root writes everywhere")
	}
	f := newFixture(t)
	f.chmod(f.modules, 0o555)
	code := f.run("--no-backup")
	if code != ExitCopy || !strings.HasSuffix(f.err, "[ng-deploy] ERROR copy failed for demo-rt.jar\n") || !strings.Contains(f.err, "cp: ") || f.exists(f.work+"/.last-deploy-sha") {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
	f = newFixture(t) // the rt jar is copied, then the ux jar fails: the deploy stops half done, as under bash
	f.write(f.deployedJar("ux"), "old")
	f.chmod(f.deployedJar("ux"), 0o444)
	t.Setenv("FAKE_OLD_VERSION", "1.0")
	t.Setenv("FAKE_NEW_VERSION", "1.1")
	code = f.run("--no-backup")
	if code != ExitCopy || !strings.HasSuffix(f.err, "[ng-deploy] ERROR copy failed for demo-ux.jar\n") || f.read(f.deployedJar("rt")) != "rtjar" {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

func TestADeployRecordsTheDeployedCommit(t *testing.T) {
	f := newFixture(t)
	f.write(f.work+"/.last-deploy-sha", "older\n")
	t.Setenv("FAKE_GIT_HEAD", "abc123")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK)
	if got := f.read(f.work + "/.last-deploy-sha"); got != "abc123\n" || !strings.HasSuffix(f.calls(), "git <rev-parse> <HEAD>\n") {
		t.Errorf(".last-deploy-sha %q, calls:\n%s", got, f.calls())
	}
}

// Kept from the bash tool: the record is not critical, so a git that fails or is absent, or a file that
// cannot be written, leaves the deploy a success.
func TestADeployWithoutACommitToRecordStillSucceeds(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GIT_HEAD_EXIT", "128")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] copy ok\n")
	if f.exists(f.work + "/.last-deploy-sha") {
		t.Error("a failed git rev-parse wrote .last-deploy-sha")
	}
	f = newFixture(t)
	if err := os.Remove(f.bin + "/git"); err != nil {
		t.Fatal(err)
	}
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] station restart required")
	if f.exists(f.work + "/.last-deploy-sha") {
		t.Error("a deploy without git wrote .last-deploy-sha")
	}
	f = newFixture(t)
	f.mkdir(f.work + "/.last-deploy-sha")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] station restart required",
		"err:ng-deploy.sh: "+f.work+"/.last-deploy-sha: Is a directory\n")
}

func TestAFailedCheckDoesNotRecordTheCommit(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "99")
	f.want(f.run("--no-backup"), ExitVerify)
	if f.exists(f.work+"/.last-deploy-sha") || strings.Contains(f.calls(), "<HEAD>") {
		t.Error("a failed verify recorded the commit")
	}
}
