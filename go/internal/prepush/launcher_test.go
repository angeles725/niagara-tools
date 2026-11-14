package prepush

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// launcherTree copies the real .githooks/pre-push and go/go-tool.sh into a throwaway checkout whose go/
// holds a module file and nothing to build, and returns the hook path and the go directory.
func launcherTree(t *testing.T) (hook, goDir string) {
	t.Helper()
	top := t.TempDir()
	hook, goDir = filepath.Join(top, ".githooks", "pre-push"), filepath.Join(top, "go")
	for dst, src := range map[string]string{hook: "../../../.githooks/pre-push", filepath.Join(goDir, "go-tool.sh"): "../../go-tool.sh"} {
		text, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, text, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module example.invalid/x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return hook, goDir
}

// launch runs the hook file with no `go` on PATH (bash and the coreutils only).
func launch(t *testing.T, hook string) (code int, stdout, stderr string) {
	t.Helper()
	for _, dir := range []string{"/usr/bin", "/bin"} {
		if _, err := os.Stat(dir + "/go"); err == nil {
			t.Skipf("%s/go exists: this host cannot stage a PATH without Go", dir)
		}
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	cmd := exec.Command(bash, hook, "origin", "https://example.invalid/repo.git")
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdin = strings.NewReader("refs/heads/x aaaa refs/heads/x bbbb\n")
	cmd.Stdout, cmd.Stderr = &out, &errb
	return exitStatus(hook, cmd.Run(), &errb), out.String(), errb.String()
}

func TestLauncherRefusesWhenTheGateCannotBeBuilt(t *testing.T) {
	hook, goDir := launcherTree(t)
	code, _, stderr := launch(t, hook)
	if code != 1 || !strings.Contains(stderr, "pre-push: go is not on PATH") ||
		!strings.Contains(stderr, "pre-push: the Go gate could not be built") || !strings.Contains(stderr, "refusing to push unchecked") {
		t.Errorf("exit %d, stderr %q; want 1 and a refusal naming the missing Go toolchain", code, stderr)
	}
	if err := os.Remove(filepath.Join(goDir, "go-tool.sh")); err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := launch(t, hook); code != 1 || !strings.Contains(stderr, "go-tool.sh missing") {
		t.Errorf("no go/go-tool.sh: exit %d, stderr %q; want a refusal", code, stderr)
	}
}

func TestLauncherRunsTheCurrentBinaryWithTheHookArgumentsAndStdin(t *testing.T) {
	hook, goDir := launcherTree(t)
	bin := filepath.Join(goDir, "bin", "pre-push")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho \"args: $*\"; cat; exit 7\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := launch(t, hook)
	if want := "args: origin https://example.invalid/repo.git\nrefs/heads/x aaaa refs/heads/x bbbb\n"; code != 7 || stdout != want || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want the binary's own 7 and %q", code, stdout, stderr, want)
	}
}
