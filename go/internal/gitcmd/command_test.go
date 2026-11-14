package gitcmd

import (
	"bytes"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd/gitcmdtest"
)

func TestMain(m *testing.M) {
	gitcmdtest.ServeFake()
	os.Exit(m.Run())
}

// TestCommandZeroValueStartsGitWhereTheProcessIs: no Dir means no -C, the subcommand is the first
// argument, the environment is the process environment and git runs in the process directory.
func TestCommandZeroValueStartsGitWhereTheProcessIs(t *testing.T) {
	fake := gitcmdtest.Install(t, gitcmdtest.Behaviour{Stdout: "out\n\n"})
	want := gitcmdtest.Environ()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	res := Command{}.Run("status", "--porcelain")
	if res.Stdout != "out\n\n" || res.Exit != 0 || !res.OK() || !res.Started() {
		t.Errorf("Run = %+v; want the stdout byte for byte, exit 0, ok, started", res)
	}
	calls := fake.Calls()
	if len(calls) != 1 || !reflect.DeepEqual(calls[0].Args, []string{"status", "--porcelain"}) ||
		!reflect.DeepEqual(calls[0].Env, want) || calls[0].Dir != cwd {
		t.Errorf("calls = %+v; want one `git status --porcelain` in %s with the process environment", calls, cwd)
	}
}

// TestCommandDirComesFirstAsDashC: a Dir is passed to git as -C, before the caller's arguments.
func TestCommandDirComesFirstAsDashC(t *testing.T) {
	fake := gitcmdtest.Install(t, gitcmdtest.Behaviour{})
	Command{Dir: "/some/dir"}.Run("rev-parse", "HEAD")
	if calls := fake.Calls(); len(calls) != 1 || !reflect.DeepEqual(calls[0].Args, []string{"-C", "/some/dir", "rev-parse", "HEAD"}) {
		t.Errorf("calls = %+v; want one `git -C /some/dir rev-parse HEAD`", calls)
	}
}

// TestCommandEnvAddsToTheProcessEnvironmentForOneRun: an Env entry reaches git, replaces the value the
// process has for that key, leaves every other variable alone and does not change the process itself.
func TestCommandEnvAddsToTheProcessEnvironmentForOneRun(t *testing.T) {
	t.Setenv("GITCMD_REPLACED", "process value")
	fake := gitcmdtest.Install(t, gitcmdtest.Behaviour{})
	var want []string
	for _, kv := range os.Environ() {
		if kv != "GITCMD_REPLACED=process value" {
			want = append(want, kv)
		}
	}
	want = append(want, "GITCMD_REPLACED=run value", "GITCMD_ADDED=1")
	sort.Strings(want)
	Command{Env: []string{"GITCMD_REPLACED=run value", "GITCMD_ADDED=1"}}.Run("fetch")
	if calls := fake.Calls(); len(calls) != 1 || !reflect.DeepEqual(calls[0].Env, want) {
		t.Errorf("git's environment\n got %q\nwant %q", calls, want)
	}
	if _, set := os.LookupEnv("GITCMD_ADDED"); set || os.Getenv("GITCMD_REPLACED") != "process value" {
		t.Error("the process environment changed")
	}
}

// TestCommandReturnsGitsExitStatus: a non-zero exit is its code with the stdout and stderr still
// delivered, and counts as started.
func TestCommandReturnsGitsExitStatus(t *testing.T) {
	for _, code := range []int{1, 5, 128} {
		gitcmdtest.Install(t, gitcmdtest.Behaviour{Stdout: "partial\n", Stderr: "fatal: scripted\n", Exit: code})
		var errb bytes.Buffer
		res := Command{Stderr: &errb}.Run("config")
		if res.Exit != code || res.OK() || !res.Started() || res.Err == nil || res.Stdout != "partial\n" || errb.String() != "fatal: scripted\n" {
			t.Errorf("exit %d: Run = %+v, stderr %q", code, res, errb.String())
		}
	}
}

// TestCommandAGitThatCannotBeStarted: no git on PATH is not an exit status: Exit is NotExited, Started
// is false and Err says why, so a caller can tell it from a git that refused.
func TestCommandAGitThatCannotBeStarted(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var errb bytes.Buffer
	res := Command{Stderr: &errb}.Run("status")
	if res.Exit != NotExited || res.OK() || res.Started() || res.Err == nil || res.Stdout != "" || errb.Len() != 0 {
		t.Errorf("Run = %+v, stderr %q; want NotExited, not started, an error, no output", res, errb.String())
	}
	if out, ok := Run(t.TempDir(), nil, "status"); out != "" || ok {
		t.Errorf("Run = %q, %v; want \"\", false", out, ok)
	}
}
