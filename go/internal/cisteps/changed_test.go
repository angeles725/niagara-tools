package cisteps

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const workflow = `jobs:
  lint:
    steps:
      - name: shellcheck (scripts, toolbelt, tests)
        run: shellcheck scripts/*.sh .githooks/* tests/*.bats
      - name: other
        run: echo shellcheck is mentioned, not run
`

func TestShellcheckGlobsAreTheArgumentsOfTheOneShellcheckStep(t *testing.T) {
	steps, err := Parse("ci.yml", []byte(workflow))
	if err != nil {
		t.Fatal(err)
	}
	got, err := ShellcheckGlobs(steps)
	if want := []string{"scripts/*.sh", ".githooks/*", "tests/*.bats"}; err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("globs %v (%v), want %v", got, err, want)
	}
	if _, err := ShellcheckGlobs(nil); err == nil {
		t.Error("a workflow with no shellcheck step must be an error, never an empty selection")
	}
}

func TestSelectSortsChangedPathsIntoTheChecks(t *testing.T) {
	globs := []string{"scripts/*.sh", ".githooks/*", "tests/*.bats"}
	got := Select(globs, []string{
		"scripts/a.sh", "scripts/lib/b.sh", ".githooks/pre-push", "tests/x.bats",
		"tests/fixtures/y.bats", "go/internal/p/p.go", "README.md",
	})
	want := Plan{
		Shellcheck: []string{"scripts/a.sh", ".githooks/pre-push", "tests/x.bats"}, // a glob never crosses a /
		Bats:       []string{"tests/x.bats"},
		Gofmt:      []string{"go/internal/p/p.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Select = %+v\nwant %+v", got, want)
	}
}

// gitRepo is a throwaway repository whose first commit is also origin/main; it returns git as a func.
func gitRepo(t *testing.T) (dir string, git func(args ...string) string) {
	t.Helper()
	dir = t.TempDir()
	git = func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_CONFIG_NOSYSTEM=1",
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.invalid", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.invalid")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	write(t, dir, ".github/workflows/ci.yml", workflow)
	write(t, dir, "scripts/old.sh", "echo old\n")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	git("update-ref", "refs/remotes/origin/main", "HEAD")
	return dir, git
}

func write(t *testing.T, dir, rel, text string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runChanged(t *testing.T, dir string, args ...string) (code int, out, errs string) {
	t.Helper()
	t.Chdir(dir)
	var o, e bytes.Buffer
	code = Main(append([]string{"changed", "--ci-file", ".github/workflows/ci.yml"}, args...), cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func TestChangedRangeMaterializesTheBytesOfTheRangeEnd(t *testing.T) {
	dir, git := gitRepo(t)
	base := git("rev-parse", "HEAD")
	write(t, dir, "scripts/new.sh", "echo committed\n")
	write(t, dir, "tests/t.bats", "@test \"a\" { :; }\n")
	git("add", "-A")
	git("commit", "-q", "-m", "change")
	git("rm", "-q", "scripts/old.sh")
	git("commit", "-q", "-m", "delete")
	head := git("rev-parse", "HEAD")
	write(t, dir, "scripts/new.sh", "echo work tree\n") // the push gates the commit, not the work tree
	tree := filepath.Join(t.TempDir(), "tree")
	code, out, errs := runChanged(t, dir, "--tree", tree, "--range", base+".."+head)
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, errs)
	}
	if want := "shellcheck\tscripts/new.sh\nshellcheck\ttests/t.bats\nbats\ttests/t.bats\n"; out != want {
		t.Errorf("plan\n%q\nwant\n%q", out, want)
	}
	if b, _ := os.ReadFile(filepath.Join(tree, "scripts/new.sh")); string(b) != "echo committed\n" {
		t.Errorf("materialized %q, want the committed bytes", b)
	}
}

func TestChangedWithoutARangeReadsTheWorkTreeAgainstOriginMain(t *testing.T) {
	dir, _ := gitRepo(t)
	write(t, dir, "scripts/old.sh", "echo edited\n")    // unstaged edit
	write(t, dir, "scripts/untracked.sh", "echo new\n") // not yet added
	tree := filepath.Join(t.TempDir(), "tree")
	code, out, errs := runChanged(t, dir, "--tree", tree)
	if code != 0 || out != "shellcheck\tscripts/old.sh\nshellcheck\tscripts/untracked.sh\n" {
		t.Fatalf("exit %d, plan %q\n%s", code, out, errs)
	}
	if b, _ := os.ReadFile(filepath.Join(tree, "scripts/old.sh")); string(b) != "echo edited\n" {
		t.Errorf("materialized %q, want the work-tree bytes", b)
	}
}

func TestChangedRefusesABadRangeAndMissingFlags(t *testing.T) {
	dir, _ := gitRepo(t)
	tree := filepath.Join(t.TempDir(), "tree")
	for _, rng := range []string{"a...b", "onlyone", "..b", "nosuch..HEAD"} {
		if code, _, _ := runChanged(t, dir, "--tree", tree, "--range", rng); code == 0 {
			t.Errorf("--range %q: exit 0", rng)
		}
	}
	if code, _, _ := runChanged(t, dir); code != ExitBad {
		t.Errorf("no --tree: exit %d, want %d", code, ExitBad)
	}
}
