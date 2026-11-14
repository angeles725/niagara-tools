package ngdeploy

import (
	"os"
	"strings"
	"testing"
)

const annotationWarning = `[ng-deploy] WARN: annotation changes detected in Java sources.
[ng-deploy] WARN: Run with --with-slotomatic to regenerate slot code, or
[ng-deploy] WARN: set SLOTOMATIC_DETECTION=off to suppress this warning.
[ng-deploy] WARN: Deploying without slotomatic may cause BComponent type errors.
`

const gitDiffCalls = "git <rev-parse> <--git-dir>\ngit <diff> <HEAD~1..HEAD> <--> <*/src/com/**/*.java>\n"

func TestAnAddedOrRemovedNiagaraAnnotationWarns(t *testing.T) {
	for diff, warns := range map[string]bool{
		"+    @NiagaraType": true, "-\t@NiagaraSingleton": true, "context\n+@NiagaraAction(name = \"x\")": true,
		"+@NiagaraProperty": true, "-@NiagaraTopic": true,
		" @NiagaraType": false, "+ // @NiagaraType": false, "+@NiagaraOther": false, "+\n@NiagaraType": false, "": false,
	} {
		f := newFixture(t)
		t.Setenv("FAKE_GIT_DIFF", diff)
		f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
		want := warnNoBackup
		if warns {
			want += annotationWarning
		}
		if f.err != want || !strings.HasPrefix(f.calls(), gitDiffCalls+"gradlew") {
			t.Errorf("diff %q: stderr %q, calls %q", diff, f.err, f.calls())
		}
	}
}

func TestWithSlotomaticSilencesTheWarning(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GIT_DIFF", "+@NiagaraType")
	f.want(f.run("--no-backup", "--no-deploy", "--with-slotomatic"), ExitOK)
	if f.err != warnNoBackup {
		t.Errorf("stderr %q", f.err)
	}
}

// Kept from the bash tool: strict mode aborts even when --with-slotomatic is given.
func TestStrictModeAbortsOnAnnotationChanges(t *testing.T) {
	for name, strict := range map[string]func(f *fixture) []string{
		"flag":            func(*fixture) []string { return []string{"--strict-slotomatic"} },
		"variable":        func(*fixture) []string { t.Setenv("SLOTOMATIC_DETECTION", "strict"); return nil },
		"with slotomatic": func(*fixture) []string { return []string{"--strict-slotomatic", "--with-slotomatic"} },
	} {
		f := newFixture(t)
		t.Setenv("FAKE_GIT_DIFF", "+@NiagaraProperty")
		code := f.run(append([]string{"--no-backup"}, strict(f)...)...)
		if code != ExitSlotomatic || f.err != warnNoBackup+"[ng-deploy] ERROR annotation changes detected; --with-slotomatic required (strict mode)\n" || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%s: exit %d, stderr %q, calls %q", name, code, f.err, f.calls())
		}
		os.Unsetenv("SLOTOMATIC_DETECTION")
	}
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy", "--strict-slotomatic"), ExitOK) // no change: strict mode passes
}

func TestDetectionOffNeverAsksGit(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GIT_DIFF", "+@NiagaraType")
	t.Setenv("SLOTOMATIC_DETECTION", "off")
	f.want(f.run("--no-backup", "--no-deploy", "--strict-slotomatic"), ExitOK)
	if f.err != warnNoBackup || strings.Contains(f.calls(), "git") {
		t.Errorf("stderr %q, calls %q", f.err, f.calls())
	}
}

func TestModeBNeverAsksGit(t *testing.T) {
	f := newFixture(t)
	t.Setenv("FAKE_GIT_DIFF", "+@NiagaraType")
	f.want(f.run("--no-backup", "--no-deploy", "--mode", "B", "--strict-slotomatic"), ExitOK)
	if f.err != warnNoBackup || strings.Contains(f.calls(), "git") {
		t.Errorf("stderr %q, calls %q", f.err, f.calls())
	}
}

// cannotDetect lists the ways the detection cannot run, with the notice bash printed for each.
var cannotDetect = map[string]struct {
	arrange func(f *fixture)
	notice  string
}{
	"no git":        {func(f *fixture) { os.Remove(f.bin + "/git") }, "[ng-deploy] NOTICE: git not found; skipping annotation change detection\n"},
	"no repository": {func(f *fixture) { f.t.Setenv("FAKE_GIT_NOREPO", "1") }, "[ng-deploy] NOTICE: not a git repository; skipping annotation change detection\n"},
	"diff fails":    {func(f *fixture) { f.t.Setenv("FAKE_GIT_DIFF_EXIT", "128") }, "[ng-deploy] NOTICE: git diff HEAD~1..HEAD failed (exit 128); skipping annotation change detection\n"},
	"unreadable baseline": {func(f *fixture) {
		f.write(f.work+"/.last-deploy-sha", recordedCommit+"\n")
		f.chmod(f.work+"/.last-deploy-sha", 0o000)
	}, ""},
}

func TestDetectionThatCannotRunIsANoticeAndTheDeployGoesOn(t *testing.T) {
	for name, tc := range cannotDetect {
		f := newFixture(t)
		tc.arrange(f)
		f.want(f.run("--no-backup", "--no-deploy"), ExitOK, "[ng-deploy] build ok\n")
		if f.err != warnNoBackup+tc.notice {
			t.Errorf("%s: stderr %q", name, f.err)
		}
	}
}

// Declared difference: under bash a detection that could not run passed strict mode like "no change".
func TestStrictModeRefusesWhenDetectionCannotRun(t *testing.T) {
	for name, tc := range cannotDetect {
		f := newFixture(t)
		tc.arrange(f)
		code := f.run("--no-backup", "--strict-slotomatic")
		want := warnNoBackup + tc.notice + "[ng-deploy] ERROR annotation change detection could not run; strict mode cannot tell whether slotomatic is needed (SLOTOMATIC_DETECTION=off skips the check)\n"
		if code != ExitSlotomatic || f.err != want || strings.Contains(f.calls(), "gradlew") {
			t.Errorf("%s: exit %d, stderr %q, calls %q", name, code, f.err, f.calls())
		}
	}
}

// recordedCommit is a .last-deploy-sha as a deploy writes it: the full object name git rev-parse HEAD prints.
const recordedCommit = "cafe01cafe01cafe01cafe01cafe01cafe01cafe"

func TestTheDiffStartsAtTheLastDeployedCommitWhenItIsOne(t *testing.T) {
	fromRecord := "git <rev-parse> <--git-dir>\ngit <cat-file> <-e> <" + recordedCommit + "^{commit}>\ngit <diff> <" + recordedCommit + "..HEAD> <--> <*/src/com/**/*.java>\n"
	sha256 := strings.Repeat("0123456789abcdef", 4)
	for _, tc := range []struct{ name, file, catFile, want string }{
		{"a commit", recordedCommit + "\n", "", fromRecord},
		{"trailing blank lines", recordedCommit + "\n\n\n", "", fromRecord},
		{"a sha-256 commit", sha256 + "\n", "", strings.ReplaceAll(fromRecord, recordedCommit, sha256)},
		{"not a commit", recordedCommit + "\n", "1", "git <rev-parse> <--git-dir>\ngit <cat-file> <-e> <" + recordedCommit + "^{commit}>\ngit <diff> <HEAD~1..HEAD> <--> <*/src/com/**/*.java>\n"},
		{"an empty file", "", "", gitDiffCalls},
	} {
		f := newFixture(t)
		f.write(f.work+"/.last-deploy-sha", tc.file)
		if tc.catFile != "" {
			t.Setenv("FAKE_GIT_CAT_FILE_EXIT", tc.catFile)
		}
		f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
		if !strings.HasPrefix(f.calls(), tc.want+"gradlew") || f.err != warnNoBackup {
			t.Errorf("%s: stderr %q, calls %q", tc.name, f.err, f.calls())
		}
	}
}

// A diff that fails after showing an annotation change is still a change: the warning, not the notice.
func TestAFailedDiffThatShowedAChangeStillWarns(t *testing.T) {
	f := newFixture(t)
	f.write(f.dir+"/do/git", "case \"$1\" in diff) echo '+@NiagaraType'; exit 2 ;; rev-parse) echo "+recordedCommit+" ;; esac\n")
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
	if f.err != warnNoBackup+annotationWarning {
		t.Errorf("stderr %q", f.err)
	}
}
