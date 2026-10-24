package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// replayText writes text as the one case of a fresh case dir (plus record.env when env is not "") and
// replays it through r.
func replayText(t *testing.T, r Runner, name, text, env string) ([]string, string) {
	t.Helper()
	r.Dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(r.Dir, name+".txt"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if env != "" {
		if err := os.WriteFile(filepath.Join(r.Dir, "record.env"), []byte(env), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return r.replay(t, filepath.Join(r.Dir, name+".txt"))
}

const head = "-- args --\n-- want --\nexit 0\n-- stderr --\n"

// TestRunnerEnvIsHermetic pins the environment a case sees: its @env plus the PATH, HOME and LC_ALL
// defaults it does not declare, sorted; no host variable reaches Getenv.
func TestRunnerEnvIsHermetic(t *testing.T) {
	t.Setenv("GOLDEN_HOST_LEAK", "leaked")
	var env []string
	var leak, home string
	r := Runner{Main: func(e, _ []string, s cli.Stdio) int {
		env, leak, home = e, s.Env("GOLDEN_HOST_LEAK"), s.Env("HOME")
		return 0
	}}
	if p, _ := replayText(t, r, "c", head+"-- @env --\nZED=1\nPATH=/x\n", ""); len(p) != 0 {
		t.Fatal(p)
	}
	want := fmt.Sprintf("[HOME=%s LC_ALL=C PATH=/x ZED=1]", home)
	if leak != "" || fmt.Sprint(env) != want || !strings.HasPrefix(home, os.TempDir()) {
		t.Errorf("env %v leak %q, want %s and no leak", env, leak, want)
	}
}

// TestRunnerTree pins the post-run tree check: the tree the run leaves must be the before tree with the
// @after and @gone sections applied, entry for entry (content, mode, symlink target, directory).
func TestRunnerTree(t *testing.T) {
	write := func(p, s string) func(string) {
		return func(d string) { os.WriteFile(filepath.Join(d, p), []byte(s), 0o644) }
	}
	for _, c := range []struct {
		name, tree, problem string
		run                 func(dir string)
	}{
		{"untouched", "-- a --\nx\n", "", func(string) {}},
		{"declared write", "-- a --\nx\n-- @after a --\ny\n", "", write("a", "y\n")},
		{"declared new file", "-- @after n/b (noeol) --\nz\n", "", func(d string) {
			os.Mkdir(filepath.Join(d, "n"), 0o755)
			write("n/b", "z")(d)
		}},
		{"declared removal", "-- a --\nx\n-- @gone a --\n", "", func(d string) { os.Remove(filepath.Join(d, "a")) }},
		{"undeclared write", "-- a --\nx\n", "changed a", write("a", "y\n")},
		{"undeclared file", "", "extra b", write("b", "")},
		{"undeclared removal", "-- a --\nx\n", "missing a", func(d string) { os.Remove(filepath.Join(d, "a")) }},
		{"declared write not done", "-- a --\nx\n-- @after a --\ny\n", "changed a", func(string) {}},
		{"mode change", "-- a (mode 755) --\nx\n", "changed a", func(d string) { os.Chmod(filepath.Join(d, "a"), 0o644) }},
		{"symlink target", "-- l (symlink) --\n@DIR@/a\n", "changed l", func(d string) {
			os.Remove(filepath.Join(d, "l"))
			os.Symlink("elsewhere", filepath.Join(d, "l"))
		}},
		{"empty dir made", "", "extra e", func(d string) { os.Mkdir(filepath.Join(d, "e"), 0o755) }},
		{"empty dir kept", "-- e (dir) --\n", "", func(string) {}},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := Runner{Main: func(_, _ []string, _ cli.Stdio) int {
				wd, _ := os.Getwd()
				c.run(wd)
				return 0
			}}
			p, _ := replayText(t, r, "c", head+c.tree, "CHDIR=1\n")
			if got := strings.Join(p, "\n"); (c.problem == "") != (got == "") || !strings.Contains(got, c.problem) {
				t.Errorf("problems %q, want one about %q", got, c.problem)
			}
		})
	}
}

// TestRunnerOutput pins the exact exit code, stdout and stderr comparison, a "(noeol)" want included.
func TestRunnerOutput(t *testing.T) {
	r := Runner{Main: func(_, _ []string, s cli.Stdio) int { fmt.Fprint(s.Out, "no newline"); return 2 }}
	if p, _ := replayText(t, r, "c", "-- args --\n-- want (noeol) --\nexit 2\nno newline\n-- stderr --\n", ""); len(p) != 0 {
		t.Errorf("exact output reported %q", p)
	}
	if p, _ := replayText(t, r, "c", "-- args --\n-- want --\nexit 2\nno newline\n-- stderr --\n", ""); len(p) == 0 {
		t.Error("a missing final newline was not reported")
	}
}

// TestRunnerPlaceholders pins @DIR@ and the Subst placeholders in arguments, env and file content, and
// their reverse in the output.
func TestRunnerPlaceholders(t *testing.T) {
	r := Runner{Subst: map[string]string{"@REPO@": "/repo/root"}, Main: func(_, args []string, s cli.Stdio) int {
		b, _ := os.ReadFile(args[0])
		fmt.Fprintf(s.Out, "%s|%s|%s\n", args[0], s.Env("R"), b)
		return 0
	}}
	text := "-- args --\n@DIR@/f\n-- want --\nexit 0\n@DIR@/f|@REPO@/x|@DIR@ @REPO@\n-- stderr --\n" +
		"-- @env --\nR=@REPO@/x\n-- f (noeol) --\n@DIR@ @REPO@\n"
	if p, _ := replayText(t, r, "c", text, ""); len(p) != 0 {
		t.Error(p)
	}
}

// TestRunnerChdir pins CHDIR=1 in record.env: the run's cwd is the case dir; without it the cwd is kept.
func TestRunnerChdir(t *testing.T) {
	var wd string
	r := Runner{Main: func(_, args []string, _ cli.Stdio) int { wd, _ = os.Getwd(); return 0 }}
	before, _ := os.Getwd()
	t.Run("chdir", func(t *testing.T) { replayText(t, r, "c", head, "CHDIR=1\n") }) // t.Chdir lasts the subtest
	if wd == before || !strings.HasPrefix(wd, os.TempDir()) {
		t.Errorf("CHDIR=1 ran in %s", wd)
	}
	t.Run("keep", func(t *testing.T) { replayText(t, r, "c", head, "") })
	if wd != before {
		t.Errorf("no CHDIR ran in %s, want %s", wd, before)
	}
}

// TestRunnerKind pins the @kind contract of a recorded case set (record.env): a recorded case names the
// pinned source and its tool versions, a truth-* case and only it says "truth: ".
func TestRunnerKind(t *testing.T) {
	env := "SOURCE=tool.sh\nCOMMIT=abc\nSHA256=00ff\n"
	ok := "-- @kind --\nrecorded from tool.sh at abc (sha256 00ff)\ntools: bash 5\n"
	r := Runner{Main: func(_, _ []string, _ cli.Stdio) int { return 0 }}
	for _, c := range []struct {
		name, kind string
		bad        bool
	}{
		{"rec", ok, false},
		{"rec", "-- @kind --\nrecorded from tool.sh at abd (sha256 00ff)\ntools: bash 5\n", true},
		{"rec", "-- @kind --\nrecorded from tool.sh at abc (sha256 00ff)\n", true},
		{"rec", "", true},
		{"rec", "-- @kind --\ntruth: fixed\n", true},
		{"truth-x", "-- @kind --\ntruth: fixed\n", false},
		{"truth-x", ok, true},
	} {
		if p, _ := replayText(t, r, c.name, head+c.kind, env); (len(p) > 0) != c.bad {
			t.Errorf("%s with %q: problems %q, want bad=%v", c.name, c.kind, p, c.bad)
		}
	}
}

// TestRunnerUnknownSection makes an unknown @ section (a stale @expect) a problem, never ignored.
func TestRunnerUnknownSection(t *testing.T) {
	r := Runner{Main: func(_, _ []string, _ cli.Stdio) int { return 0 }}
	if p, _ := replayText(t, r, "c", head+"-- @expect a --\nx\n", ""); len(p) == 0 {
		t.Error("@expect was accepted")
	}
}

// TestRunnerSetup pins the Setup hook: what it adds to env reaches the run, and the entries it made
// ("p/" = p and all below it) are not part of the tree comparison.
func TestRunnerSetup(t *testing.T) {
	r := Runner{
		Setup: func(t *testing.T, dir string, env map[string]string) []string {
			env["ADDED"] = dir
			os.MkdirAll(filepath.Join(dir, ".tools", "x"), 0o755)
			os.WriteFile(filepath.Join(dir, "own"), nil, 0o644)
			return []string{".tools/", "own"}
		},
		Main: func(_, _ []string, s cli.Stdio) int { fmt.Fprintln(s.Out, s.Env("ADDED")); return 0 },
	}
	if p, _ := replayText(t, r, "c", "-- args --\n-- want --\nexit 0\n@DIR@\n-- stderr --\n", ""); len(p) != 0 {
		t.Error(p)
	}
}

// TestRunnerLockedEntries pins mode 000 entries: they are materialized locked (the run sees no read
// access), compared, and cleaned up.
func TestRunnerLockedEntries(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads mode 000")
	}
	var readErr error
	r := Runner{Main: func(_, _ []string, _ cli.Stdio) int {
		_, readErr = os.ReadFile("d/f")
		return 0
	}}
	p, _ := replayText(t, r, "c", head+"-- d (dir) (mode 000) --\n-- d/f --\nx\n-- u (mode 000) --\nsecret\n", "CHDIR=1\n")
	if len(p) != 0 || readErr == nil {
		t.Errorf("problems %q, read error %v (want none and a permission error)", p, readErr)
	}
}

// TestRunnerOrder pins deterministic problem order: tree problems come in path order.
func TestRunnerOrder(t *testing.T) {
	r := Runner{Main: func(_, _ []string, _ cli.Stdio) int {
		for _, f := range []string{"z", "b", "m"} {
			os.WriteFile(f, nil, 0o644)
		}
		return 0
	}}
	p, _ := replayText(t, r, "c", head, "CHDIR=1\n")
	if got := strings.Join(p, "\n"); !strings.Contains(got, "extra b") || strings.Index(got, "extra b") > strings.Index(got, "extra m") ||
		strings.Index(got, "extra m") > strings.Index(got, "extra z") {
		t.Errorf("problems %q, want b, m, z in order", got)
	}
}

func TestCountProblem(t *testing.T) {
	for _, c := range []struct {
		n, want int
		atLeast bool
		bad     bool
	}{{3, 3, false, false}, {4, 3, false, true}, {2, 3, false, true}, {4, 3, true, false}, {2, 3, true, true}} {
		if got := countProblem(c.n, c.want, c.atLeast); (got != "") != c.bad {
			t.Errorf("countProblem(%d, %d, %v) = %q, want bad=%v", c.n, c.want, c.atLeast, got, c.bad)
		}
	}
}

func TestRootVerdict(t *testing.T) {
	locked := []string{"z", "a"}
	fail := "running as root: a cannot be made unreadable; run as a normal user or set " + RootOptIn + "=1 to skip"
	for _, c := range []struct {
		uid    int
		optIn  string
		locked []string
		skip   bool
		msg    string
	}{
		{1000, "", locked, false, ""},
		{0, "", nil, false, ""},
		{0, "", locked, false, fail},
		{0, "yes", locked, false, fail},
		{0, "1", locked, true, "running as root: mode 000 does not block reads (" + RootOptIn + "=1)"},
	} {
		if skip, msg := rootVerdict(c.uid, c.optIn, c.locked); skip != c.skip || msg != c.msg {
			t.Errorf("rootVerdict(%d, %q, %q) = %v, %q; want %v, %q", c.uid, c.optIn, c.locked, skip, msg, c.skip, c.msg)
		}
	}
}
