package golden

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// RootOptIn lets a run as root skip (instead of fail) the cases that hold an owner-unreadable entry.
const RootOptIn = "GOLDEN_ALLOW_ROOT_SKIP"

// Runner replays a directory of golden cases hermetically (record/README.md has the case format).
// A case's tree is written under a fresh dir, the command runs with only the case's environment, and the
// exit code, stdout, stderr and the whole tree the run leaves must be exactly what the case records.
type Runner struct {
	// Dir holds the cases (*.txt) and, for a set recorded by record/record.sh, its record.env.
	Dir string
	// Cases is the exact number of case files (a minimum with AtLeast).
	Cases   int
	AtLeast bool
	// Subst maps more placeholders to their values (e.g. "@REPO@" to the repository root).
	Subst map[string]string
	// Setup prepares the case dir after its tree is written and before the run; it may add to env. It
	// returns the entries it made, which the tree comparison skips ("p/" skips p and all below it).
	Setup func(t *testing.T, dir string, env map[string]string) []string
	// Main runs the command: env is the case's whole environment, sorted, and s.Getenv reads only it.
	Main func(env, args []string, s cli.Stdio) int
}

// defaultEnv is the environment a case gets for each of these variables it does not declare (record.sh gives
// the bash run the same). The private recorders before the Runner left HOME and LC_ALL unset; the preflight,
// commissioning-verify and new-retro cases re-recorded byte-identical under these defaults, and none of those
// ports reads HOME or LC_ALL itself (commissioning-verify adds LC_ALL=C to its lints' env on its own).
var defaultEnv = map[string]string{"PATH": "/usr/bin:/bin", "HOME": "@DIR@", "LC_ALL": "C"}

// Run replays every case of r.Dir as a subtest, in name order, one at a time.
func (r Runner) Run(t *testing.T) {
	t.Helper()
	dir, err := filepath.Abs(r.dir()) // a CHDIR=1 case runs in its tree, and -update writes the case after it
	if err != nil {
		t.Fatal(err)
	}
	paths, _ := filepath.Glob(filepath.Join(dir, "*.txt"))
	if msg := countProblem(len(paths), r.Cases, r.AtLeast); msg != "" {
		t.Fatal(msg)
	}
	defer syscall.Umask(syscall.Umask(0o022)) // files the run creates get the modes the recorder's runs gave them
	for _, p := range paths {
		t.Run(strings.TrimSuffix(filepath.Base(p), ".txt"), func(t *testing.T) {
			problems, skip := r.replay(t, p)
			for _, msg := range problems {
				t.Error(msg)
			}
			if skip != "" && len(problems) == 0 {
				t.Skip(skip)
			}
		})
	}
}

func (r Runner) dir() string {
	if r.Dir == "" {
		return "testdata"
	}
	return r.Dir
}

// countProblem says why n case files are not the want the runner expects ("" when they are).
func countProblem(n, want int, atLeast bool) string {
	if n == want || (atLeast && n > want) {
		return ""
	}
	if atLeast {
		return fmt.Sprintf("found %d golden cases, want at least %d", n, want)
	}
	return fmt.Sprintf("found %d golden cases, want exactly %d", n, want)
}

// rootVerdict decides a case holding the owner-unreadable entries locked for a run as uid with RootOptIn
// set to optIn: no message runs it; otherwise skip says whether to skip (optIn "1") or fail, and the
// message names the first locked entry in path order.
func rootVerdict(uid int, optIn string, locked []string) (skip bool, msg string) {
	if uid != 0 || len(locked) == 0 {
		return false, ""
	}
	if optIn == "1" {
		return true, "running as root: mode 000 does not block reads (" + RootOptIn + "=1)"
	}
	first := locked[0]
	for _, l := range locked {
		first = min(first, l)
	}
	return false, "running as root: " + first + " cannot be made unreadable; run as a normal user or set " + RootOptIn + "=1 to skip"
}
