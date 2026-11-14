// Package gitcmdtest gives the tests of a git caller a fake `git` that records exactly what it was
// started with (argv, environment, working directory) and answers with a scripted stdout, stderr and
// exit code. The fake is the test binary itself under the name git, so nothing is rewritten by a shell
// on the way: a package that uses it calls ServeFake first thing in its TestMain.
package gitcmdtest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	fakeName      = "git"
	behaviourFile = "behaviour.json"
	callsDir      = "calls"
)

// Behaviour is what the fake prints and the code it exits with.
type Behaviour struct {
	Stdout, Stderr string
	Exit           int
}

// Call is one start of the fake: its arguments after argv[0], its environment sorted, its directory.
type Call struct {
	Args []string
	Env  []string
	Dir  string
}

// Fake is a scripted git that is the only directory of PATH while the test runs.
type Fake struct {
	t   testing.TB
	dir string
}

// Install puts the fake first and alone on PATH for the rest of the test, answering with b.
func Install(t testing.TB, b Behaviour) *Fake {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f := &Fake{t: t, dir: t.TempDir()}
	if err := os.Mkdir(filepath.Join(f.dir, callsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := linkOrCopy(self, filepath.Join(f.dir, fakeName)); err != nil {
		t.Fatal(err)
	}
	f.Answer(b)
	t.Setenv("PATH", f.dir)
	return f
}

// Answer changes what the next calls print and exit with.
func (f *Fake) Answer(b Behaviour) {
	f.t.Helper()
	raw, err := json.Marshal(b)
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.dir, behaviourFile), raw, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// Calls are the starts recorded so far, in order.
func (f *Fake) Calls() []Call {
	f.t.Helper()
	var calls []Call
	for n := 1; ; n++ {
		raw, err := os.ReadFile(callPath(f.dir, n))
		if os.IsNotExist(err) {
			return calls
		}
		var c Call
		if err == nil {
			err = json.Unmarshal(raw, &c)
		}
		if err != nil {
			f.t.Fatal(err)
		}
		calls = append(calls, c)
	}
}

// Environ is the environment a child of this process inherits right now, sorted like Call.Env.
func Environ() []string {
	env := os.Environ()
	sort.Strings(env)
	return env
}

func callPath(dir string, n int) string {
	return filepath.Join(dir, callsDir, fmt.Sprintf("%04d.json", n))
}

func linkOrCopy(from, to string) error {
	if os.Link(from, to) == nil {
		return nil
	}
	raw, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return os.WriteFile(to, raw, 0o755)
}
