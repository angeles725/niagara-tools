package hookinstall

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const templateText = "#!/usr/bin/env bash\n# kit template\nexit 0\n"

// fixture is a throwaway repository and a throwaway kit tree (scripts/, toolbelt/, templates/); git
// reads a scratch global config, so no host hooksPath leaks in and none is written.
type fixture struct {
	t                                 *testing.T
	base, repo, kitRoot, globalConfig string
	launcher                          string // <kitRoot>/scripts/install-client-hooks.sh
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %v: %v\n%s", dir, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, text string, perm os.FileMode) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
	return path
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, base: base, repo: filepath.Join(base, "repo"), kitRoot: filepath.Join(base, "kit-checkout"),
		globalConfig: filepath.Join(base, "gitconfig")}
	t.Setenv("GIT_CONFIG_GLOBAL", f.globalConfig)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CEILING_DIRECTORIES", base)
	git(t, base, "init", "-q", "-b", "main", f.repo)
	f.launcher = write(t, filepath.Join(f.kitRoot, "scripts", "install-client-hooks.sh"), "#!/usr/bin/env bash\n", 0o755)
	write(t, f.template(), templateText, 0o755)
	if err := os.MkdirAll(f.toolbelt(), 0o755); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *fixture) toolbelt() string {
	return filepath.Join(f.kitRoot, "build-n4-module-kit", "toolbelt")
}

func (f *fixture) template() string {
	return filepath.Join(f.kitRoot, "build-n4-module-kit", "templates", "client-pre-push")
}

func (f *fixture) hook() string { return filepath.Join(f.repo, ".githooks", "pre-push") }

// local is the repository's own value of a config key ("" when unset).
func (f *fixture) local(key string) string {
	out, _ := exec.Command("git", "-C", f.repo, "config", "--local", "--get-all", key).Output()
	return strings.TrimSpace(string(out))
}

// hooks runs install-hooks with dir as the working directory.
func (f *fixture) hooks(dir string, args ...string) (int, string, string) {
	f.t.Chdir(dir)
	var out, errb bytes.Buffer
	code := Main(args, cli.Stdio{Out: &out, Err: &errb})
	return code, out.String(), errb.String()
}

// client runs install-client-hooks as the fixture's launcher.
func (f *fixture) client(args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	code := ClientMain(args, cli.Stdio{Out: &out, Err: &errb, Getenv: func(name string) string {
		if name == installtools.LauncherEnv {
			return f.launcher
		}
		return os.Getenv(name)
	}})
	return code, out.String(), errb.String()
}

func expect(t *testing.T, name string, code int, out, errText string, wantCode int, wantOut, wantErr string) {
	t.Helper()
	if code != wantCode || out != wantOut || errText != wantErr {
		t.Errorf("%s:\n got exit %d\n stdout %q\n stderr %q\nwant exit %d\n stdout %q\n stderr %q",
			name, code, out, errText, wantCode, wantOut, wantErr)
	}
}

func absent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil {
		t.Errorf("%s exists; nothing should have been written", path)
	}
}
