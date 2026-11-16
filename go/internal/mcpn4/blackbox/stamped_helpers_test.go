package blackbox

import (
	"debug/buildinfo"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// stampedBinary builds a trivial Go program committed in a scratch git repository, so it carries
// a VCS stamp, and returns its path and the revision stamped in it. It skips without go or git.
func stampedBinary(t *testing.T) (path, revision string) {
	t.Helper()
	for _, tool := range []string{"go", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not on PATH: %v", tool, err)
		}
	}
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module stamped\n\ngo 1.21\n")
	write("main.go", "package main\n\nfunc main() {}\n")
	run := func(name string, args ...string) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOFLAGS=-buildvcs=true", "GOWORK=off", "GOTOOLCHAIN=local",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init", "-q")
	run("git", "add", ".")
	run("git", "-c", "user.name=t", "-c", "user.email=t@example.invalid", "commit", "-q", "-m", "stamped")
	path = filepath.Join(dir, "stamped")
	run("go", "build", "-o", path, ".")
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			revision = s.Value
		}
	}
	if revision == "" {
		t.Fatalf("no vcs.revision in %v", info.Settings)
	}
	return path, revision
}

// writeScratch is a scratch file holding body.
func writeScratch(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// replaceBinary renames a copy of src over bin, as an install does (writing a running executable
// in place fails), and sets its mtime to now plus offset.
func replaceBinary(t *testing.T, bin, src string, offset time.Duration) {
	t.Helper()
	body, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	tmp := bin + ".new"
	if err := os.WriteFile(tmp, body, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(tmp, bin); err != nil {
		t.Fatal(err)
	}
	touch(t, bin, offset)
}
