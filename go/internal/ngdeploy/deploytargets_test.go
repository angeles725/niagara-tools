package ngdeploy

import (
	"path/filepath"
	"strings"
	"testing"
)

// The tests of --target: after a deploy that went through, ng-deploy records what it deployed in
// <module root>/.deploy-targets, the file build.sh --target reads (go/internal/buildmodule/deploytarget.go).
// The module root is the directory of GRADLEW_PATH, the one that holds the module directory; the working
// directory (where .last-deploy-sha goes) is another one in the fixture.

const (
	headCommit    = "deadbeef00deadbeef00deadbeef00deadbeef00" // what the stub git prints for HEAD
	noTargetLine  = "[ng-deploy] record: not written to .deploy-targets; to record this deploy for build.sh --target, deploy with --target <name> (a name of your own, one per station)\n"
	oldCommitLine = "st1 demo@1.0=0123456789abcdef0123456789abcdef01234567"
)

// targets is the .deploy-targets file of the fixture's module root.
func (f *fixture) targets() string { return f.src + "/.deploy-targets" }

// deployTo is a mode C deploy with the new jar at version 1.2, to the target.
func (f *fixture) deployTo(target string, extra ...string) int {
	f.t.Setenv("FAKE_NEW_VERSION", "1.2")
	return f.run(append([]string{"--no-backup", "--mode", "C", "--target", target}, extra...)...)
}

// A deploy to a target with no .deploy-targets yet creates the file with the one line, and says so.
func TestATargetDeployWritesTheLineToANewFile(t *testing.T) {
	f := newFixture(t)
	f.want(f.deployTo("st1"), ExitOK, "[ng-deploy] record: st1 demo@1.2="+headCommit+" written to "+f.targets()+" (build.sh --target st1 reads it)\n")
	if got := f.read(f.targets()); got != "st1 demo@1.2="+headCommit+"\n" {
		t.Errorf(".deploy-targets holds %q", got)
	}
	if f.exists(f.work+"/.deploy-targets") || f.read(f.work+"/.last-deploy-sha") != headCommit+"\n" {
		t.Error("the record belongs to the module root, and .last-deploy-sha stays in the working directory")
	}
	if left, _ := filepath.Glob(f.src + "/.deploy-targets*"); len(left) != 1 {
		t.Errorf("the atomic write left %v", left)
	}
}

// A line for another target or another module is not the line of this deploy: it is appended after them, and
// comments, blank lines and the other lines stay as they were (even a file with no final newline).
func TestATargetDeployAppendsAndKeepsTheRestOfTheFile(t *testing.T) {
	for name, tail := range map[string]string{"a final newline": "\n", "no final newline": ""} {
		f := newFixture(t)
		keep := "# what each station runs\n\nst2 demo@3.0=cafe0123cafe0123  # trailing note\nst1 other@1.0=cafe0123cafe0123"
		f.write(f.targets(), keep+tail)
		f.want(f.deployTo("st1"), ExitOK)
		if got := f.read(f.targets()); got != keep+"\nst1 demo@1.2="+headCommit+"\n" {
			t.Errorf("%s: .deploy-targets holds %q", name, got)
		}
	}
}

// The line of the same target and module is replaced where it stands, and a repeat of it is dropped.
func TestATargetDeployReplacesTheLineOfItsTargetAndModule(t *testing.T) {
	f := newFixture(t)
	f.write(f.targets(), "# stations\nst2 demo@3.0=cafe0123cafe0123\n"+oldCommitLine+" # was 1.0\n# last\nst1 demo@0.9=0123456789abcdef\n")
	f.want(f.deployTo("st1"), ExitOK)
	want := "# stations\nst2 demo@3.0=cafe0123cafe0123\nst1 demo@1.2=" + headCommit + "\n# last\n"
	if got := f.read(f.targets()); got != want {
		t.Errorf(".deploy-targets holds %q, want %q", got, want)
	}
}

// Without --target nothing is written, a file that is there stays as it is, and one line says how to record.
func TestADeployWithoutATargetWritesNothingAndSaysHow(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, noTargetLine)
	if f.exists(f.targets()) || strings.Count(f.out, "record:") != 1 {
		t.Errorf(".deploy-targets exists: %v; stdout:\n%s", f.exists(f.targets()), f.out)
	}
	f = newFixture(t)
	f.write(f.targets(), oldCommitLine+"\n")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK, noTargetLine)
	if got := f.read(f.targets()); got != oldCommitLine+"\n" {
		t.Errorf(".deploy-targets holds %q", got)
	}
}

// A run that deployed nothing, or whose copy or record did not go through, leaves the file as it was.
func TestARunThatDidNotDeployWritesNothing(t *testing.T) {
	for _, tc := range []struct {
		name  string
		code  int
		set   map[string]string
		extra []string
	}{
		{"the build fails", ExitBuild, map[string]string{"FAKE_GRADLEW_EXIT": "1"}, nil},
		{"the gate fails", ExitVerify, map[string]string{"FAKE_VM_EXIT": "1"}, nil},
		{"--no-deploy", ExitOK, nil, []string{"--no-deploy"}},
		{"the commit cannot be read", ExitRecord, map[string]string{"FAKE_GIT_HEAD_EXIT": "128", "SLOTOMATIC_DETECTION": "warn"}, nil},
	} {
		f := newFixture(t)
		for k, v := range tc.set {
			t.Setenv(k, v)
		}
		f.write(f.targets(), oldCommitLine+"\n")
		if code := f.deployTo("st1", tc.extra...); code != tc.code || strings.Contains(f.out, "record:") && tc.code != ExitOK {
			t.Errorf("%s: exit %d, stdout:\n%sstderr:\n%s", tc.name, code, f.out, f.err)
		}
		if got := f.read(f.targets()); got != oldCommitLine+"\n" || strings.Contains(f.out, "written to") {
			t.Errorf("%s: .deploy-targets holds %q", tc.name, got)
		}
	}
}

// A deploy to a target whose line cannot be written, or has no commit or version to put in it, has copied the
// jars by then: exit 41 and one ERROR line that says what is missing.
func TestATargetThatCannotBeRecordedExits41AfterTheCopy(t *testing.T) {
	for _, tc := range []struct{ name, why string }{
		{"no commit", "the commit is unknown: git rev-parse HEAD failed (exit 128)"},
		{"no version", "the vendorVersion of demo-rt.jar cannot be read"},
		{"a record that is a directory", "cannot write " + "SRC/.deploy-targets"},
	} {
		f := newFixture(t)
		switch tc.name {
		case "no commit":
			t.Setenv("FAKE_GIT_HEAD_EXIT", "128")
			t.Setenv("SLOTOMATIC_DETECTION", "off")
		case "a record that is a directory":
			f.mkdir(f.targets())
		}
		code := f.deployTo("st1")
		if tc.name == "no version" {
			t.Setenv("FAKE_NEW_VERSION", "")
			code = f.run("--no-backup", "--mode", "C", "--target", "st1")
		}
		want := "[ng-deploy] ERROR jars copied, but the deploy was not recorded for target st1 in " + f.targets() + ": " + strings.ReplaceAll(tc.why, "SRC", f.src)
		if code != ExitRecord || !strings.Contains(f.err, want) || f.read(f.deployedJar("rt")) != "rtjar" {
			t.Errorf("%s: exit %d, stderr %q", tc.name, code, f.err)
		}
		if strings.Contains(f.out, "done:") {
			t.Errorf("%s: a run that could not record prints no closing lines:\n%s", tc.name, f.out)
		}
	}
}

// Mode B deploys no -rt jar, so the commit of the target's -rt schema does not change: no line, and a line saying why.
func TestAUxOnlyDeployDoesNotMoveTheRecordedCommit(t *testing.T) {
	f := newFixture(t)
	f.write(f.targets(), oldCommitLine+"\n")
	t.Setenv("FAKE_NEW_VERSION", "1.2")
	f.want(f.run("--no-backup", "--mode", "B", "--target", "st1"), ExitOK,
		"[ng-deploy] record: st1 not updated in .deploy-targets: mode B deploys no -rt jar, so the -rt commit it names stays what the station runs\n")
	if got := f.read(f.targets()); got != oldCommitLine+"\n" {
		t.Errorf(".deploy-targets holds %q", got)
	}
}

// A target name is one word of letters, digits, ".", "_" and "-" (a blank or "#" would break the line); a bad
// one, or a --target without a name, is refused before anything runs.
func TestATargetNameThatBreaksTheLineIsRefusedBeforeAnyCommand(t *testing.T) {
	for _, bad := range []string{"two words", "a#b", "-x", "a@b", "a=b", ".dot"} {
		f := newFixture(t)
		code := f.run("--no-backup", "--mode", "C", "--target", bad)
		want := "[ng-deploy] ERROR --target \"" + bad + "\" is not a target name (letters, digits, '.', '_' and '-', starting with a letter or a digit)\n"
		if code != ExitEnv || f.err != want || f.calls() != "" || f.exists(f.targets()) {
			t.Errorf("%q: exit %d, stderr %q, calls %q", bad, code, f.err, f.calls())
		}
	}
	f := newFixture(t)
	if code := f.run("--target"); code != ExitEnv || f.err != "[ng-deploy] ERROR --target requires an argument\n" {
		t.Errorf("no name: exit %d, stderr %q", code, f.err)
	}
}

// The target may be named in the env file, as every option is; the flag wins.
func TestATargetFromTheEnvFileIsRecordedAndTheFlagWins(t *testing.T) {
	f := newFixture(t)
	f.set("DEPLOY_TARGET", "st9")
	f.want(f.deployTo("st1"), ExitOK)
	if got := f.read(f.targets()); got != "st1 demo@1.2="+headCommit+"\n" {
		t.Errorf("flag over file: %q", got)
	}
	f = newFixture(t)
	f.set("DEPLOY_TARGET", "st9")
	t.Setenv("FAKE_NEW_VERSION", "1.2")
	f.want(f.run("--no-backup", "--mode", "C"), ExitOK)
	if got := f.read(f.targets()); got != "st9 demo@1.2="+headCommit+"\n" {
		t.Errorf("file: %q", got)
	}
}
