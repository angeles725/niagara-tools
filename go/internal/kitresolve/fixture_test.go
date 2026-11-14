package kitresolve

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fixture is a scratch bare origin, the seed clone that pushes to it and a scratch home: every
// candidate is a clone under base, so no real checkout is read or touched.
type fixture struct {
	t                        *testing.T
	base, origin, seed, home string
	kitMain, defaultTop, dev string // candidate checkouts (created by clone)
	env                      map[string]string
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %v: %v\n%s", dir, args, err, out)
	}
	return strings.TrimSpace(string(out))
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
	f := &fixture{t: t, base: base, origin: filepath.Join(base, "origin.git"), seed: filepath.Join(base, "seed"),
		home: filepath.Join(base, "home"), dev: filepath.Join(base, "dev")}
	f.kitMain = filepath.Join(f.home, "modulos_niagara_n4", "niagara-tools-worktrees", "kit-main")
	f.defaultTop = filepath.Join(f.home, "modulos_niagara_n4", "niagara-tools")
	f.env = map[string]string{"HOME": f.home}
	kit := filepath.Join(f.seed, kitDirName)
	if err := os.MkdirAll(kit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kit, "METHODOLOGY.md"), []byte("methodology\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, base, "init", "-q", "--bare", "-b", "main", f.origin)
	git(t, base, "init", "-q", "-b", "main", f.seed)
	git(t, f.seed, "add", "-A")
	git(t, f.seed, "commit", "-q", "-m", "c1")
	git(t, f.seed, "remote", "add", "origin", f.origin)
	git(t, f.seed, "push", "-q", "origin", "main")
	return f
}

// clone makes a clean checkout of origin/main at top.
func (f *fixture) clone(top string) {
	f.t.Helper()
	if err := os.MkdirAll(filepath.Dir(top), 0o755); err != nil {
		f.t.Fatal(err)
	}
	git(f.t, f.base, "clone", "-q", f.origin, top)
}

// advance gives origin/main one more commit.
func (f *fixture) advance() {
	f.t.Helper()
	git(f.t, f.seed, "commit", "-q", "--allow-empty", "-m", "next")
	git(f.t, f.seed, "push", "-q", "origin", "main")
}

// localCommit puts one commit that is not on origin/main on top's detached HEAD.
func (f *fixture) localCommit(top string) {
	f.t.Helper()
	git(f.t, top, "checkout", "-q", "--detach")
	git(f.t, top, "commit", "-q", "--allow-empty", "-m", "local")
}

// dirty edits a tracked file of top.
func (f *fixture) dirty(top string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(top, kitDirName, "METHODOLOGY.md"), []byte("edited\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) head(top string) string { return git(f.t, top, "rev-parse", "HEAD") }

func (f *fixture) short7(top, rev string) string {
	return git(f.t, top, "rev-parse", "--short=7", rev)
}

func (f *fixture) resolve(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(name string) string { return f.env[name] }})
	return code, out.String(), errb.String()
}

// okLine is the success line for a kit under top at its current HEAD.
func (f *fixture) okLine(top, fresh string) string {
	return "KIT=" + filepath.Join(top, kitDirName) + " SHA=" + git(f.t, top, "rev-parse", "--short", "HEAD") + " (" + fresh + ", clean)\n"
}

func remedyKitMain(top string) string {
	return "refresh: git -C " + top + " checkout --detach origin/main"
}

func remedyEnv(top string) string {
	return "update " + top + " to origin/main, or unset BUILD_N4_KIT to use the kit-main worktree"
}

func (f *fixture) remedyDefault(top string) string {
	return "use the kit-main worktree: git -C " + top + " worktree add --detach " + f.kitMain + " origin/main"
}

func expect(t *testing.T, name string, code int, out, errText string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode || out != wantOut || errText != wantErr {
		t.Errorf("%s:\n got exit %d\n stdout %q\n stderr %q\nwant exit %d\n stdout %q\n stderr %q",
			name, code, out, errText, wantCode, wantOut, wantErr)
	}
}
