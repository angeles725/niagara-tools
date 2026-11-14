package installtools

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

// TestGitStatusPinsWhatGitIsStartedWith: one start of `git -C <dir> <args>` in the process directory
// with the process environment untouched; stdout comes back byte for byte, stderr goes to errTo and
// the status is git's own, on success and on failure.
func TestGitStatusPinsWhatGitIsStartedWith(t *testing.T) {
	t.Setenv("GITSTATUS_PIN_MARKER", "kept as it is")
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range []gitcmdtest.Behaviour{
		{Stdout: "line one\n\nlast\n\n", Stderr: "", Exit: 0},
		{Stdout: "partial\n", Stderr: "fatal: scripted\n", Exit: 5},
		{Stdout: "", Stderr: "fatal: not a work tree\n", Exit: 128},
	} {
		fake := gitcmdtest.Install(t, b)
		want := gitcmdtest.Environ()
		var errb strings.Builder
		out, status := GitStatus(dir, &errb, "config", "--local", "a key", "")
		if out != b.Stdout || status != b.Exit || errb.String() != b.Stderr {
			t.Errorf("exit %d: GitStatus = %q, %d, stderr %q; want %q, %d, %q", b.Exit, out, status, errb.String(), b.Stdout, b.Exit, b.Stderr)
		}
		calls := fake.Calls()
		if len(calls) != 1 {
			t.Fatalf("exit %d: git started %d times, want once", b.Exit, len(calls))
		}
		if wantArgs := []string{"-C", dir, "config", "--local", "a key", ""}; !reflect.DeepEqual(calls[0].Args, wantArgs) {
			t.Errorf("exit %d: argv %q, want %q", b.Exit, calls[0].Args, wantArgs)
		}
		if !reflect.DeepEqual(calls[0].Env, want) {
			t.Errorf("exit %d: git's environment differs from the process environment\n got %q\nwant %q", b.Exit, calls[0].Env, want)
		}
		if calls[0].Dir != cwd {
			t.Errorf("exit %d: git started in %s, want the process directory %s", b.Exit, calls[0].Dir, cwd)
		}
	}
}

// TestGitStatusPinsDroppedStderr: with no errTo the stderr of git is dropped and the status still returned.
func TestGitStatusPinsDroppedStderr(t *testing.T) {
	gitcmdtest.Install(t, gitcmdtest.Behaviour{Stdout: "out\n", Stderr: "noise\n", Exit: 4})
	if out, status := GitStatus(t.TempDir(), nil, "config", "x", "y"); out != "out\n" || status != 4 {
		t.Errorf("GitStatus = %q, %d; want \"out\\n\", 4", out, status)
	}
}

// TestGitStatusPinsAGitThatCannotBeStarted: no git on PATH is status 127 with one line naming the
// reason on errTo, and 127 in silence without errTo; never git's own 128 or a 0.
func TestGitStatusPinsAGitThatCannotBeStarted(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	var errb strings.Builder
	out, status := GitStatus(t.TempDir(), &errb, "rev-parse", "--show-toplevel")
	if want := "git: exec: \"git\": executable file not found in $PATH\n"; out != "" || status != 127 || errb.String() != want {
		t.Errorf("GitStatus = %q, %d, stderr %q; want \"\", 127, %q", out, status, errb.String(), want)
	}
	if out, status := GitStatus(t.TempDir(), nil, "rev-parse"); out != "" || status != 127 {
		t.Errorf("no errTo: GitStatus = %q, %d; want \"\", 127", out, status)
	}
}
