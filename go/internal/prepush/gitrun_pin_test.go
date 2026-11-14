package prepush

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd/gitcmdtest"
)

func TestMain(m *testing.M) {
	gitcmdtest.ServeFake()
	os.Exit(m.Run())
}

// TestGitRunPinsWhatGitIsStartedWith: the gate starts `git <args>` with the subcommand as the first
// argument (no -C), in the process directory, with the process environment untouched. The stdout comes
// back without its trailing newlines only, stderr goes to errTo, and ok is "git exited 0".
func TestGitRunPinsWhatGitIsStartedWith(t *testing.T) {
	t.Setenv("GITRUN_PIN_MARKER", "kept as it is")
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		answer  gitcmdtest.Behaviour
		wantOut string
	}{
		{gitcmdtest.Behaviour{Stdout: "\n a\n\nb \n\n\n"}, "\n a\n\nb "},
		{gitcmdtest.Behaviour{Stdout: "kept on failure\n", Stderr: "fatal: scripted\n", Exit: 1}, "kept on failure"},
		{gitcmdtest.Behaviour{Stderr: "fatal: bad revision\n", Exit: 128}, ""},
	} {
		fake := gitcmdtest.Install(t, c.answer)
		want := gitcmdtest.Environ()
		var errb strings.Builder
		out, ok := gitOut(&errb, "diff", "--name-only", "a..b", "--", "a path/*.md")
		if out != c.wantOut || ok != (c.answer.Exit == 0) || errb.String() != c.answer.Stderr {
			t.Errorf("exit %d: gitOut = %q, %v, stderr %q; want %q, %v, %q",
				c.answer.Exit, out, ok, errb.String(), c.wantOut, c.answer.Exit == 0, c.answer.Stderr)
		}
		if gitOK(nil, "merge-base", "a", "b") != (c.answer.Exit == 0) {
			t.Errorf("exit %d: gitOK disagrees with the exit status", c.answer.Exit)
		}
		calls := fake.Calls()
		if len(calls) != 2 {
			t.Fatalf("exit %d: git started %d times, want twice", c.answer.Exit, len(calls))
		}
		wantArgs := [][]string{{"diff", "--name-only", "a..b", "--", "a path/*.md"}, {"merge-base", "a", "b"}}
		for i, call := range calls {
			if !reflect.DeepEqual(call.Args, wantArgs[i]) {
				t.Errorf("exit %d: argv %q, want %q", c.answer.Exit, call.Args, wantArgs[i])
			}
			if !reflect.DeepEqual(call.Env, want) {
				t.Errorf("exit %d: git's environment differs from the process environment\n got %q\nwant %q", c.answer.Exit, call.Env, want)
			}
			if call.Dir != cwd {
				t.Errorf("exit %d: git started in %s, want the process directory %s", c.answer.Exit, call.Dir, cwd)
			}
		}
	}
}

// TestGitRunPinsAGitThatCannotBeStarted: no git on PATH is a failure with nothing on stdout and
// nothing written to errTo, so the caller refuses with its own message.
func TestGitRunPinsAGitThatCannotBeStarted(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var errb strings.Builder
	if out, ok := gitOut(&errb, "rev-parse", "--show-toplevel"); out != "" || ok || errb.Len() != 0 {
		t.Errorf("gitOut = %q, %v, stderr %q; want \"\", false, no stderr", out, ok, errb.String())
	}
	if gitOK(&errb, "rev-parse") {
		t.Error("gitOK = true with no git on PATH")
	}
}
