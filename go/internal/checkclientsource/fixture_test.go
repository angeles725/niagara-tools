package checkclientsource

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const wantUsage = "usage: check-client-source.sh [--declared <path>@<branch>@<commit>|unknown] <module-root>\n" +
	"       check-client-source.sh --build-state <BUILD-STATE.md> --module <MOD> <module-root>\n"

// fixture is a repository `r` on branch main with two commits and a module root Group/Mod.
type fixture struct {
	t               *testing.T
	base, top, root string // the directory holding r, r itself, r/Group/Mod
	first, second   string // the two commit ids
}

func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	for k, v := range map[string]string{
		"GIT_CONFIG_GLOBAL": os.DevNull, "GIT_CONFIG_NOSYSTEM": "1",
		"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@example.invalid",
		"GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@example.invalid",
	} {
		t.Setenv(k, v)
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, base: base, top: filepath.Join(base, "r")}
	f.root = filepath.Join(f.top, "Group", "Mod")
	write(t, filepath.Join(f.root, "a.txt"), "a\n")
	gitOut(t, f.top, "init", "-q", "-b", "main")
	f.first = f.commit(f.top, "one")
	write(t, filepath.Join(f.root, "b.txt"), "b\n")
	f.second = f.commit(f.top, "two")
	return f
}

func (f *fixture) commit(repo, message string) string {
	gitOut(f.t, repo, "add", "-A")
	gitOut(f.t, repo, "commit", "-q", "-m", message)
	return gitOut(f.t, repo, "rev-parse", "HEAD")
}

func run(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

func expect(t *testing.T, name string, args []string, code int, out, errText string) {
	t.Helper()
	if c, o, e := run(args...); c != code || o != out || e != errText {
		t.Errorf("%s: exit %d\nstdout %q\nstderr %q\nwant exit %d\nstdout %q\nstderr %q", name, c, o, e, code, out, errText)
	}
}
