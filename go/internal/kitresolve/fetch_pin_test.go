package kitresolve

import (
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd/gitcmdtest"
)

const fetchPinPrompt = "GIT_TERMINAL_PROMPT"

func TestMain(m *testing.M) {
	gitcmdtest.ServeFake()
	os.Exit(m.Run())
}

// fetchPinWantEnv is the process environment with the credential prompt switched off, sorted.
func fetchPinWantEnv() []string {
	var env []string
	for _, kv := range os.Environ() {
		if len(kv) <= len(fetchPinPrompt) || kv[:len(fetchPinPrompt)+1] != fetchPinPrompt+"=" {
			env = append(env, kv)
		}
	}
	env = append(env, fetchPinPrompt+"=0")
	sort.Strings(env)
	return env
}

// TestFetchPinsWhatGitIsStartedWith: the fetch is one start of `git -C <top> fetch -q origin main`
// whose environment is the process environment plus GIT_TERMINAL_PROMPT=0, whether the caller had the
// variable unset or set to something else; the caller's own value is back afterwards; the result is
// whether git exited 0, whatever it printed.
func TestFetchPinsWhatGitIsStartedWith(t *testing.T) {
	top := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name          string
		promptSet     bool
		prompt        string
		answer        gitcmdtest.Behaviour
		wantFetchedOK bool
	}{
		{"prompt unset, fetch succeeds", false, "", gitcmdtest.Behaviour{Stdout: "noise\n", Stderr: "noise\n"}, true},
		{"prompt set to 1, fetch fails", true, "1", gitcmdtest.Behaviour{Stdout: "noise\n", Stderr: "fatal: offline\n", Exit: 128}, false},
		{"prompt set empty, fetch fails with 1", true, "", gitcmdtest.Behaviour{Exit: 1}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(fetchPinPrompt, c.prompt) // registers the restore
			if !c.promptSet {
				os.Unsetenv(fetchPinPrompt)
			}
			fake := gitcmdtest.Install(t, c.answer)
			want := fetchPinWantEnv()
			if got := (repo{top: top}).fetch(); got != c.wantFetchedOK {
				t.Errorf("fetch() = %v, want %v", got, c.wantFetchedOK)
			}
			calls := fake.Calls()
			if len(calls) != 1 {
				t.Fatalf("git started %d times, want once", len(calls))
			}
			if wantArgs := []string{"-C", top, "fetch", "-q", "origin", "main"}; !reflect.DeepEqual(calls[0].Args, wantArgs) {
				t.Errorf("argv %q, want %q", calls[0].Args, wantArgs)
			}
			if !reflect.DeepEqual(calls[0].Env, want) {
				t.Errorf("git's environment is not the process environment plus %s=0\n got %q\nwant %q", fetchPinPrompt, calls[0].Env, want)
			}
			if calls[0].Dir != cwd {
				t.Errorf("git started in %s, want the process directory %s", calls[0].Dir, cwd)
			}
			if now, set := os.LookupEnv(fetchPinPrompt); set != c.promptSet || now != c.prompt {
				t.Errorf("after the fetch %s is %q (set %v); want the caller's %q (set %v)", fetchPinPrompt, now, set, c.prompt, c.promptSet)
			}
		})
	}
}

// TestFetchPinsAGitThatCannotBeStarted: no git on PATH is a failed fetch, not a panic.
func TestFetchPinsAGitThatCannotBeStarted(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if (repo{top: t.TempDir()}).fetch() {
		t.Error("fetch() = true with no git on PATH")
	}
}
