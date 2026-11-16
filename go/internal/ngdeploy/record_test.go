package ngdeploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	noticeUnknownRecordLine = "[ng-deploy] NOTICE: .last-deploy-sha names a commit git does not know; annotation changes are looked for since HEAD~1 only\n"
	noticeBadRecordLine     = "[ng-deploy] NOTICE: .last-deploy-sha does not hold a full commit id; annotation changes are looked for since HEAD~1 only\n"
	recordErrorPrefix       = "[ng-deploy] ERROR jars copied, but the deployed commit was not recorded in .last-deploy-sha: "
	recordErrorSuffix       = " (the next annotation check cannot start at this deploy; SLOTOMATIC_DETECTION=off makes the record optional)\n"
)

// A record that is not a full hex object name never reaches git: a leading "-" would be read as an option.
func TestARecordThatIsNotAFullObjectNameNeverReachesGit(t *testing.T) {
	for name, record := range map[string]string{
		"an option": "--output=/tmp/x", "a short name": "cafe01", "a ref": "HEAD", "upper case": strings.ToUpper(recordedCommit),
		"a range": recordedCommit + "..HEAD", "two lines": recordedCommit + "\n" + recordedCommit, "39 digits": recordedCommit[1:],
		"41 digits": recordedCommit + "a", "63 digits": recordedCommit + recordedCommit[:23],
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
		{"the record cannot be written any more", func(f *fixture) { // it could before the copy: git puts a directory in its place
			f.remove(f.work + "/.last-deploy-sha")
			f.write(f.dir+"/do/git", `case "$1 $2" in "rev-parse HEAD") mkdir .last-deploy-sha; echo `+recordedCommit+` ;; "rev-parse --git-dir") echo .git ;; esac`)
		}, "cannot write WORK/.last-deploy-sha: Is a directory", ""},
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

// A record that cannot be written is found while the station still holds the old jars: exit 10 after the
// checks and before the copy, naming the file and the reason, and nothing is left behind by the look.
func TestARecordThatCannotBeWrittenIsFoundBeforeTheCopy(t *testing.T) {
	for _, tc := range []struct {
		name    string
		arrange func(f *fixture)
		why     string
		root    bool // root can write it all the same
	}{
		{"a directory", func(f *fixture) { f.mkdir(f.work + "/.last-deploy-sha") }, "Is a directory", false},
		{"a read-only file", func(f *fixture) { f.write(f.work+"/.last-deploy-sha", ""); f.chmod(f.work+"/.last-deploy-sha", 0o444) }, "Permission denied", true},
		{"no record in a read-only directory", func(f *fixture) { f.chmod(f.work, 0o555) }, "Permission denied", true},
	} {
		if tc.root && os.Getuid() == 0 {
			continue
		}
		f := newFixture(t)
		f.deployed()
		t.Setenv("FAKE_OLD_VERSION", "1.0")
		t.Setenv("FAKE_NEW_VERSION", "1.1")
		tc.arrange(f)
		code := f.run("--no-backup")
		want := "[ng-deploy] ERROR the deployed commit cannot be recorded in .last-deploy-sha (cannot write " + f.work + "/.last-deploy-sha: " + tc.why +
			"); nothing was copied (fix the file, or SLOTOMATIC_DETECTION=off makes the record optional)\n"
		if code != ExitEnv || !strings.HasSuffix(f.err, want) || !strings.HasSuffix(f.out, "[ng-deploy] gate ok\n") || strings.Contains(f.calls(), "<HEAD>") {
			t.Errorf("%s: exit %d, stdout:\n%sstderr %q", tc.name, code, f.out, f.err)
		}
		os.Chmod(f.work, 0o755)
		if f.read(f.deployedJar("rt")) != "old-rt" || f.read(f.deployedJar("ux")) != "old-ux" || (tc.name == "no record in a read-only directory" && f.exists(f.work+"/.last-deploy-sha")) {
			t.Errorf("%s: the station holds rt %q ux %q", tc.name, f.read(f.deployedJar("rt")), f.read(f.deployedJar("ux")))
		}
	}
}

// The look before the copy leaves a working directory without a record as it found it, and a record that
// is there untouched, when the deploy then stops for another reason.
func TestTheLookAtTheRecordBeforeTheCopyChangesNothing(t *testing.T) {
	for _, record := range []string{"", recordedCommit + "\n"} {
		f := newFixture(t)
		if record != "" {
			f.write(f.work+"/.last-deploy-sha", record)
		}
		failRename(t, "/demo-rt.jar")
		f.want(f.run("--no-backup"), ExitCopy)
		if f.read(f.work+"/.last-deploy-sha") != record || f.exists(f.work+"/.last-deploy-sha") != (record != "") {
			t.Errorf("record %q: .last-deploy-sha holds %q", record, f.read(f.work+"/.last-deploy-sha"))
		}
	}
}

// A full object name git does not know (another clone's commit, a history rewritten since) is asked about
// once and then left out of the diff, which starts at HEAD~1 and still finds a change there.
func TestARecordGitDoesNotKnowIsNotHandedToTheDiff(t *testing.T) {
	f := newFixture(t)
	f.write(f.work+"/.last-deploy-sha", recordedCommit+"\n")
	t.Setenv("FAKE_GIT_CAT_FILE_EXIT", "1")
	t.Setenv("FAKE_GIT_DIFF", "+@NiagaraType")
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] build ok\n")
	calls := "git <rev-parse> <--git-dir>\ngit <cat-file> <-e> <" + recordedCommit + "^{commit}>\ngit <diff> <HEAD~1..HEAD> <--> <*/src/com/**/*.java>\ngradlew"
	if f.err != warnNoBackup+noticeUnknownRecordLine+annotationWarning || !strings.HasPrefix(f.calls(), calls) {
		t.Errorf("stderr %q, calls %q", f.err, f.calls())
	}
}

// The look before the copy answers what the write after it will do, links included: a record that is a
// symbolic link to a file not there yet is written through the link, as it was before the look existed.
func TestARecordThatIsALinkToAFileNotThereYetIsWrittenThroughTheLink(t *testing.T) {
	f := newFixture(t)
	if err := os.Symlink(f.dir+"/elsewhere", f.work+"/.last-deploy-sha"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_GIT_HEAD", recordedCommit)
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, "[ng-deploy] copy ok\n")
	if to, _ := os.Readlink(f.work + "/.last-deploy-sha"); f.err != warnNoBackup || f.read(f.dir+"/elsewhere") != recordedCommit+"\n" || to != f.dir+"/elsewhere" {
		t.Errorf("stderr %q, the link leads to %q holding %q", f.err, to, f.read(f.dir+"/elsewhere"))
	}

	f = newFixture(t) // the deploy stops at the copy: the file the look created through the link is gone again
	if err := os.Symlink(f.dir+"/elsewhere", f.work+"/.last-deploy-sha"); err != nil {
		t.Fatal(err)
	}
	failRename(t, "/demo-rt.jar")
	f.want(f.run("--no-backup", "--mode", "C"), ExitCopy)
	if to, _ := os.Readlink(f.work + "/.last-deploy-sha"); f.exists(f.dir+"/elsewhere") || to != f.dir+"/elsewhere" {
		t.Errorf("the look left %q behind, or changed the link (%q)", f.dir+"/elsewhere", to)
	}
}

// A link into a directory that is not there cannot be written, before the copy or after it: exit 10 with
// the reason the write itself would give.
func TestARecordThatIsALinkIntoAMissingDirectoryIsFoundBeforeTheCopy(t *testing.T) {
	f := newFixture(t)
	if err := os.Symlink(f.dir+"/nodir/x", f.work+"/.last-deploy-sha"); err != nil {
		t.Fatal(err)
	}
	code := f.run("--no-backup", "--mode", "C")
	want := warnNoBackup + "[ng-deploy] ERROR the deployed commit cannot be recorded in .last-deploy-sha (cannot write " + f.work + "/.last-deploy-sha: No such file or directory" +
		"); nothing was copied (fix the file, or SLOTOMATIC_DETECTION=off makes the record optional)\n"
	if code != ExitEnv || f.err != want || f.exists(f.deployedJar("rt")) {
		t.Errorf("exit %d, stderr %q", code, f.err)
	}
}

// A module directory that is not a git repository has no commit to record: the deploy that went through
// ends 0 with a NOTICE, not exit 41 (retro 2026-11-15-coordination-stretch-deltas D9, user decision
// 2026-11-15). A repository whose HEAD cannot be read is still exit 41 (above).
func TestADeployOutsideGitEndsZeroWithANotice(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GIT_NOREPO", "1")
	t.Setenv("FAKE_GIT_HEAD_EXIT", "128")
	code := f.run("--no-backup", "--mode", "C")
	want := "[ng-deploy] NOTICE: not a git repository; the deployed commit is not recorded in .last-deploy-sha: the next annotation check has no starting point\n"
	f.want(code, ExitOK, "[ng-deploy] copy ok\n[ng-deploy] station restart required (Java classes deployed)\n", "err:"+want)
	if strings.Contains(f.err, "ERROR") || f.exists(f.work+"/.last-deploy-sha") || f.read(f.deployedJar("rt")) != "rtjar" {
		t.Errorf("stderr %q", f.err)
	}
}

// Only "not a git repository" with no .git above is outside git: a repository git refuses to open (dubious
// ownership on a WSL mount, a corrupt .git) is still exit 41 with the ERROR (review R1/R3/R4
// outside-git-misclassifies-unreadable-repo).
func TestARepositoryGitCannotOpenIsStillExit41(t *testing.T) {
	for name, arrange := range map[string]func(f *fixture){
		"dubious ownership": func(f *fixture) {
			f.t.Setenv("FAKE_GIT_NOREPO_SAYS", "fatal: detected dubious ownership in repository at '/mnt/c/x'")
		},
		"a corrupt .git above": func(f *fixture) { f.mkdir(f.work + "/.git") },
	} {
		f := newFixture(t)
		t.Setenv("FAKE_GIT_NOREPO", "1")
		t.Setenv("FAKE_GIT_HEAD_EXIT", "128")
		arrange(f)
		code := f.run("--no-backup", "--mode", "C")
		if code != ExitRecord || !strings.HasSuffix(f.err, recordErrorPrefix+"git rev-parse HEAD failed (exit 128)"+recordErrorSuffix) || strings.Contains(f.err, "NOTICE: not a git repository; the deployed") {
			t.Errorf("%s: exit %d, stderr %q", name, code, f.err)
		}
	}
}

// The same two cases with the host's git: a plain directory is outside git (exit 0, NOTICE); one holding a
// .git that is not a repository is exit 41.
func TestOutsideGitWithTheHostGit(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("no git on this host")
	}
	for _, corrupt := range []bool{false, true} {
		f := newFixture(t)
		f.remove(f.bin + "/git")
		if err := os.Symlink(git, f.bin+"/git"); err != nil {
			t.Fatal(err)
		}
		t.Setenv("SLOTOMATIC_DETECTION", "warn")
		t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(f.dir))
		if corrupt {
			f.mkdir(f.work + "/.git")
		}
		code := f.run("--no-backup", "--mode", "C")
		want, notice := ExitOK, true
		if corrupt {
			want, notice = ExitRecord, false
		}
		if code != want || strings.Contains(f.err, "NOTICE: not a git repository; the deployed commit") != notice {
			t.Errorf("corrupt %v: exit %d, stderr %q", corrupt, code, f.err)
		}
	}
}
