package ngdeploy

import (
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
		if code != ExitOK || !strings.Contains(f.out, tc.last+"[ng-deploy] done: ") || f.err != warnNoBackup { // the closing lines follow (nextsteps_test.go)
			t.Errorf("mode %s: exit %d, stdout:\n%sstderr %q", tc.mode, code, f.out, f.err)
		}
		if (f.read(f.deployedJar("rt")) == "rtjar") != tc.rt || (f.read(f.deployedJar("ux")) == "uxjar") != tc.ux {
			t.Errorf("mode %s: station holds rt %q ux %q", tc.mode, f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")))
		}
	}
}

// A deploy that could record its commit is unchanged: the full object name git printed, one line, exit 0.
func TestADeployRecordsTheDeployedCommit(t *testing.T) {
	f := newFixture(t)
	f.write(f.work+"/.last-deploy-sha", strings.Repeat("0", 40)+"\n") // the record of an older deploy
	t.Setenv("FAKE_GIT_HEAD", recordedCommit)
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK)
	if got := f.read(f.work + "/.last-deploy-sha"); got != recordedCommit+"\n" || !strings.HasSuffix(f.calls(), "git <rev-parse> <HEAD>\n") || f.err != warnNoBackup {
		t.Errorf(".last-deploy-sha %q, stderr %q, calls:\n%s", got, f.err, f.calls())
	}
}

func TestAFailedCheckDoesNotRecordTheCommit(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_UNZIP_TYPES", "99")
	f.want(f.run("--no-backup"), ExitVerify)
	if f.exists(f.work+"/.last-deploy-sha") || strings.Contains(f.calls(), "<HEAD>") {
		t.Error("a failed verify recorded the commit")
	}
}
