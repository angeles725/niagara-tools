package ngdeploy

import (
	"os"
	"testing"
)

func TestHelpPrintsTheUsageAndExits0(t *testing.T) {
	f := newFixture(t)
	f.noEnvFile = true
	if code := f.run("--help"); code != ExitOK || f.out != usage || f.err != "" {
		t.Errorf("--help: exit %d, stdout %q, stderr %q", code, f.out, f.err)
	}
	// the flags are read in order: --help ends the run before a later flag or the mode is checked
	if code := f.run("--mode", "Z", "--help", "--banana"); code != ExitOK || f.out != usage {
		t.Errorf("--mode Z --help: exit %d, stdout %q", code, f.out)
	}
	if f.calls() != "" {
		t.Errorf("--help ran a command: %s", f.calls())
	}
}

func TestVersionPrintsTheVersionFileBesideTheScripts(t *testing.T) {
	f := newFixture(t)
	f.noEnvFile = true
	if code := f.run("--version"); code != ExitOK || f.out != "9.8.7\n" {
		t.Errorf("--version: exit %d, stdout %q", code, f.out)
	}
	if err := os.Remove(f.dir + "/repo/VERSION"); err != nil {
		t.Fatal(err)
	}
	if code := f.run("--version"); code != ExitOK || f.out != "unknown\n" {
		t.Errorf("--version without a VERSION file: exit %d, stdout %q", code, f.out)
	}
}

func TestUnknownFlagPrintsTheErrorThenTheUsageAndExits10(t *testing.T) {
	f := newFixture(t)
	for _, arg := range []string{"--banana", "positional"} {
		code := f.run("--no-backup", arg, "--help")
		if code != ExitEnv || f.out != usage || f.err != "[ng-deploy] ERROR unknown flag: "+arg+"\n" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", arg, code, f.out, f.err)
		}
	}
}

func TestAFlagWithoutItsValueExits10(t *testing.T) {
	f := newFixture(t)
	for _, flag := range []string{"--mode", "--env-file", "--keep"} {
		code := f.run(flag)
		if code != ExitEnv || f.out != "" || f.err != "[ng-deploy] ERROR "+flag+" requires an argument\n" {
			t.Errorf("%s: exit %d, stdout %q, stderr %q", flag, code, f.out, f.err)
		}
	}
}

func TestAnInvalidModeExits10BeforeTheEnvFileIsRead(t *testing.T) {
	f := newFixture(t)
	f.write(f.env, "echo not an assignment\n")
	for _, mode := range []string{"Z", "a", ""} {
		code := f.run("--mode", mode)
		if code != ExitEnv || f.err != "[ng-deploy] ERROR invalid --mode '"+mode+"'; valid values are A, B, C\n" {
			t.Errorf("mode %q: exit %d, stderr %q", mode, code, f.err)
		}
	}
}

func TestTheRetiredCompanionFlagIsAcceptedAndChangesNothing(t *testing.T) {
	f := newFixture(t)
	f.want(f.run("--no-backup", "--no-deploy"), ExitOK)
	out, errs, calls := f.out, f.err, f.calls()
	f.reset()
	f.want(f.run("--no-backup", "--i-know-what-im-doing", "--no-deploy"), ExitOK)
	if f.out != out || f.err != errs || f.calls() != calls {
		t.Errorf("--i-know-what-im-doing changed the run:\n%s%s%s", f.out, f.err, f.calls())
	}
}
