package installtools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestGitConfigReadsUnsetLastValueAndFailure: unset is "", several values give the last, the local
// scope ignores the global file, and a directory that is no repository is an error, not "unset".
func TestGitConfigReadsUnsetLastValueAndFailure(t *testing.T) {
	global := write(t, filepath.Join(t.TempDir(), "gitconfig"), "[core]\n\thooksPath = /global/hooks\n", 0o644)
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	if v, err := GitConfig(repo, nil, Local, "core.hookspath"); v != "" || err != nil {
		t.Errorf("local, unset = %q, %v", v, err)
	}
	if v, err := GitConfig(repo, nil, Effective, "core.hookspath"); v != "/global/hooks" || err != nil {
		t.Errorf("effective = %q, %v; want the global value", v, err)
	}
	git("config", "--local", "--add", "core.hooksPath", "first")
	git("config", "--local", "--add", "core.hooksPath", "a b\nc")
	if v, err := GitConfig(repo, nil, Local, "core.hookspath"); v != "a b\nc" || err != nil {
		t.Errorf("local, two values = %q, %v; want the last", v, err)
	}
	if v, err := GitConfig(t.TempDir(), nil, Local, "core.hookspath"); err == nil {
		t.Errorf("no repository = %q, nil; want an error", v)
	}
}

// TestGitStatusReturnsGitsOwnExitStatus: 0 with the stdout, git's status on a failure (128 outside a
// repository) with its message on errTo.
func TestGitStatusReturnsGitsOwnExitStatus(t *testing.T) {
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	var errb strings.Builder
	if out, status := GitStatus(dir, &errb, "rev-parse", "--show-toplevel"); status != 128 || out != "" || !strings.Contains(errb.String(), "fatal:") {
		t.Errorf("outside a repository: %q, status %d, stderr %q; want status 128 and git's fatal line", out, status, errb.String())
	}
	if out, status := GitStatus(dir, nil, "init", "-q", "-b", "main"); status != 0 || out != "" {
		t.Errorf("git init: %q, status %d", out, status)
	}
	if out, status := GitStatus(dir, nil, "symbolic-ref", "--short", "HEAD"); status != 0 || out != "main\n" {
		t.Errorf("symbolic-ref: %q, status %d", out, status)
	}
	if _, status := GitStatus(dir, nil, "config", "--local", "--unset", "core.hooksPath"); status != 5 {
		t.Errorf("unset of an unset key: status %d, want git's 5", status)
	}
}
