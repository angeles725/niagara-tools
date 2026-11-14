package ngdeploy

import (
	"strings"
	"testing"
)

const (
	noticeBadRecordLine = "[ng-deploy] NOTICE: .last-deploy-sha does not hold a full commit id; annotation changes are looked for since HEAD~1 only\n"
	recordErrorPrefix   = "[ng-deploy] ERROR jars copied, but the deployed commit was not recorded in .last-deploy-sha: "
	recordErrorSuffix   = " (the next annotation check cannot start at this deploy; SLOTOMATIC_DETECTION=off makes the record optional)\n"
)

// A record that is not a full hex object name never reaches git: a leading "-" would be read as an option.
func TestARecordThatIsNotAFullObjectNameNeverReachesGit(t *testing.T) {
	for name, record := range map[string]string{
		"an option": "--output=/tmp/x", "a short name": "cafe01", "a ref": "HEAD", "upper case": strings.ToUpper(recordedCommit),
		"a range": recordedCommit + "..HEAD", "two lines": recordedCommit + "\n" + recordedCommit, "39 digits": recordedCommit[1:],
		"a blank before": " " + recordedCommit, "a carriage return": recordedCommit + "\r",
	} {
		f := newFixture(t)
		f.write(f.work+"/.last-deploy-sha", record+"\n")
		f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] build ok\n")
		if f.err != warnNoBackup+noticeBadRecordLine || !strings.HasPrefix(f.calls(), gitDiffCalls+"gradlew") {
			t.Errorf("%s: stderr %q, calls %q", name, f.err, f.calls())
		}
		f = newFixture(t) // strict mode cannot tell what changed since a deploy it cannot name
		f.write(f.work+"/.last-deploy-sha", record+"\n")
		code := f.run("--no-backup", "--strict-slotomatic")
		if code != ExitSlotomatic || !strings.HasPrefix(f.err, warnNoBackup+noticeBadRecordLine+"[ng-deploy] ERROR annotation change detection could not run;") || strings.Contains(f.calls(), "cat-file") || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%s, strict: exit %d, stderr %q, calls %q", name, code, f.err, f.calls())
		}
	}
}

// The jars are in the station by then: the run says so, tells the operator what to restart, and exits 41.
func TestADeployThatCannotRecordItsCommitExits41AfterTheCopy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		why     string
		left    string // .last-deploy-sha after the run
	}{
		{"git rev-parse fails", func(f *fixture) { f.t.Setenv("FAKE_GIT_HEAD_EXIT", "128") }, "git rev-parse HEAD failed (exit 128)", "older\n"},
		{"no git", func(f *fixture) { f.remove(f.bin + "/git") }, "git rev-parse HEAD failed (exit 127)", "older\n"},
		{"git prints nothing", func(f *fixture) { f.t.Setenv("FAKE_GIT_HEAD", "") }, "git rev-parse HEAD printed no full commit id", "older\n"},
		{"git prints a short name", func(f *fixture) { f.t.Setenv("FAKE_GIT_HEAD", "abc123") }, "git rev-parse HEAD printed no full commit id", "older\n"},
		{"the record cannot be written", func(f *fixture) { f.remove(f.work + "/.last-deploy-sha"); f.mkdir(f.work + "/.last-deploy-sha") },
			"cannot write WORK/.last-deploy-sha: Is a directory", ""},
	} {
		f := newFixture(t)
		f.write(f.work+"/.last-deploy-sha", "older\n")
		t.Setenv("SLOTOMATIC_DETECTION", "warn")
		tc.arrange(f)
		code := f.run("--no-backup", "--mode", "C")
		wantErr := recordErrorPrefix + strings.ReplaceAll(tc.why, "WORK", f.work) + recordErrorSuffix
		if code != ExitRecord || !strings.HasSuffix(f.err, wantErr) || strings.Count(f.err, "ERROR") != 1 {
			t.Errorf("%s: exit %d, stderr %q", tc.name, code, f.err)
		}
		if !strings.HasSuffix(f.out, "[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n") || f.read(f.deployedJar("rt")) != "rtjar" {
			t.Errorf("%s: stdout:\n%s", tc.name, f.out)
		}
		if got := f.read(f.work + "/.last-deploy-sha"); got != tc.left {
			t.Errorf("%s: .last-deploy-sha holds %q, want %q", tc.name, got, tc.left)
		}
	}
}

// With the detection off nothing reads the record: it stays best effort, as before.
func TestTheRecordIsOptionalWhenTheDetectionIsOff(t *testing.T) {
	f := newFixture(t)
	t.Setenv("SLOTOMATIC_DETECTION", "off")
	t.Setenv("FAKE_GIT_HEAD_EXIT", "128")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] station restart required (Java classes deployed)\n")
	if f.err != warnNoBackup || f.exists(f.work+"/.last-deploy-sha") {
		t.Errorf("stderr %q", f.err)
	}
	f = newFixture(t)
	t.Setenv("SLOTOMATIC_DETECTION", "off")
	f.mkdir(f.work + "/.last-deploy-sha")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] station restart required (Java classes deployed)\n")
	if f.err != warnNoBackup+"ng-deploy.sh: "+f.work+"/.last-deploy-sha: Is a directory\n" {
		t.Errorf("stderr %q", f.err)
	}
}
