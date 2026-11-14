package installtools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

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

func env(vars map[string]string) cli.Stdio {
	return cli.Stdio{Getenv: func(name string) string { return vars[name] }}
}

// TestLauncherLocatesTheRepositoryAndReadsItsHeader: the repository is the parent of the launcher's
// directory, and the header is the comment block after the shebang, up to the first code line.
func TestLauncherLocatesTheRepositoryAndReadsItsHeader(t *testing.T) {
	root := t.TempDir()
	script := write(t, filepath.Join(root, "scripts", "tool.sh"),
		"#!/usr/bin/env bash\n# tool.sh — one line.\n#\n#   indented\nset -u\n# not the header\n", 0o755)
	l, err := LauncherFrom(env(map[string]string{LauncherEnv: script}))
	if err != nil {
		t.Fatal(err)
	}
	if l.RepoRoot() != root || l.Sibling("other.sh") != filepath.Join(root, "scripts", "other.sh") {
		t.Errorf("RepoRoot %q, Sibling %q", l.RepoRoot(), l.Sibling("other.sh"))
	}
	if h, err := l.Header(); err != nil || h != "tool.sh — one line.\n\n  indented\n" {
		t.Errorf("Header = %q, %v", h, err)
	}
}

// TestLauncherRefusesAnUnsetOrRelativePath: without the launcher there is no repository to work for.
func TestLauncherRefusesAnUnsetOrRelativePath(t *testing.T) {
	for _, value := range []string{"", "scripts/tool.sh"} {
		if _, err := LauncherFrom(env(map[string]string{LauncherEnv: value})); err == nil ||
			!strings.Contains(err.Error(), LauncherEnv) {
			t.Errorf("LauncherFrom(%q) = %v; want an error naming %s", value, err, LauncherEnv)
		}
	}
	if _, err := (Launcher{Path: filepath.Join(t.TempDir(), "gone.sh")}).Header(); err == nil {
		t.Error("Header of a missing launcher: no error")
	}
}

// TestSHA256AndSameContent: the digest is sha256sum's, and an unreadable file is an error for both.
func TestSHA256AndSameContent(t *testing.T) {
	dir := t.TempDir()
	a := write(t, filepath.Join(dir, "a"), "abc", 0o644)
	b := write(t, filepath.Join(dir, "b"), "abc", 0o644)
	c := write(t, filepath.Join(dir, "c"), "abd", 0o644)
	const abc = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got, err := SHA256(a); err != nil || got != abc {
		t.Errorf("SHA256 = %q, %v", got, err)
	}
	if same, err := SameContent(a, b); err != nil || !same {
		t.Errorf("SameContent(equal) = %v, %v", same, err)
	}
	if same, err := SameContent(a, c); err != nil || same {
		t.Errorf("SameContent(different) = %v, %v", same, err)
	}
	gone := filepath.Join(dir, "gone")
	if _, err := SHA256(gone); err == nil {
		t.Error("SHA256 of a missing file: no error")
	}
	for _, pair := range [][2]string{{a, gone}, {gone, a}} {
		if same, err := SameContent(pair[0], pair[1]); err == nil || same {
			t.Errorf("SameContent(%v) = %v, %v; want an error", pair, same, err)
		}
	}
}

// TestFileTestsTellAbsentFromUnreadable: a missing path (or one through a regular file) is "no"; a
// path that cannot be examined is an error.
func TestFileTestsTellAbsentFromUnreadable(t *testing.T) {
	dir := t.TempDir()
	file := write(t, filepath.Join(dir, "d", "f"), "x", 0o644)
	if ok, err := IsFile(file); !ok || err != nil {
		t.Errorf("IsFile(file) = %v, %v", ok, err)
	}
	if ok, err := IsFile(dir); ok || err != nil {
		t.Errorf("IsFile(dir) = %v, %v", ok, err)
	}
	if ok, err := IsDir(filepath.Dir(file)); !ok || err != nil {
		t.Errorf("IsDir(dir) = %v, %v", ok, err)
	}
	for _, path := range []string{filepath.Join(dir, "gone"), filepath.Join(file, "below")} {
		if ok, err := IsFile(path); ok || err != nil {
			t.Errorf("IsFile(%s) = %v, %v; want false, nil", path, ok, err)
		}
		if ok, err := Exists(path); ok || err != nil {
			t.Errorf("Exists(%s) = %v, %v; want false, nil", path, ok, err)
		}
		if ok, err := IsDir(path); ok || err != nil {
			t.Errorf("IsDir(%s) = %v, %v; want false, nil", path, ok, err)
		}
	}
	if os.Getuid() == 0 {
		t.Skip("root reads through a mode-000 directory")
	}
	if err := os.Chmod(filepath.Dir(file), 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Dir(file), 0o755) })
	if ok, err := IsFile(file); ok || err == nil {
		t.Errorf("IsFile(unreadable) = %v, %v; want an error", ok, err)
	}
	if ok, err := Exists(file); ok || err == nil {
		t.Errorf("Exists(unreadable) = %v, %v; want an error", ok, err)
	}
	if ok, err := IsDir(file); ok || err == nil {
		t.Errorf("IsDir(unreadable) = %v, %v; want an error", ok, err)
	}
}

// TestIsExecutableFile: a regular file with an execute bit, never a directory or a plain file.
func TestIsExecutableFile(t *testing.T) {
	dir := t.TempDir()
	exe := write(t, filepath.Join(dir, "x"), "#!/bin/sh\n", 0o755)
	plain := write(t, filepath.Join(dir, "p"), "#!/bin/sh\n", 0o644)
	if !IsExecutableFile(exe) || IsExecutableFile(plain) || IsExecutableFile(dir) || IsExecutableFile(filepath.Join(dir, "gone")) {
		t.Error("IsExecutableFile: want true only for the executable regular file")
	}
}

// TestCopyIsCp: a new target takes the source mode under the umask; an existing one keeps its mode and
// its inode, and is written through a symlink.
func TestCopyIsCp(t *testing.T) {
	old := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(old) })
	dir := t.TempDir()
	src := write(t, filepath.Join(dir, "src"), "new\n", 0o775)
	dst := filepath.Join(dir, "dst")
	if err := Copy(src, dst); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(dst); st.Mode().Perm() != 0o755 {
		t.Errorf("new target mode %v, want 0755", st.Mode().Perm())
	}
	real := write(t, filepath.Join(dir, "real"), "old content, longer\n", 0o600)
	link := filepath.Join(dir, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := Copy(src, link); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Lstat(link); st.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink was replaced")
	}
	data, _ := os.ReadFile(real)
	if st, _ := os.Stat(real); string(data) != "new\n" || st.Mode().Perm() != 0o600 {
		t.Errorf("existing target: content %q mode %v; want the new content and 0600", data, st.Mode().Perm())
	}
	if err := Copy(filepath.Join(dir, "gone"), dst); err == nil {
		t.Error("Copy of a missing source: no error")
	}
	if err := Copy(src, filepath.Join(dir, "no-dir", "dst")); err == nil {
		t.Error("Copy into a missing directory: no error")
	}
}

// TestAddExecHonoursTheUmask: `chmod +x` adds only the execute bits the umask allows.
func TestAddExecHonoursTheUmask(t *testing.T) {
	old := syscall.Umask(0o027)
	t.Cleanup(func() { syscall.Umask(old) })
	file := write(t, filepath.Join(t.TempDir(), "f"), "x", 0o644)
	if err := AddExec(file); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(file); st.Mode().Perm() != 0o754 {
		t.Errorf("mode %v, want 0754", st.Mode().Perm())
	}
	if err := AddExec(file + ".gone"); err == nil {
		t.Error("AddExec of a missing file: no error")
	}
}

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
