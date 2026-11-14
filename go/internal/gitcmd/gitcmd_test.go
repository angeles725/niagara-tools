package gitcmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// repo is an empty repository kept off the host's git configuration.
func repo(t *testing.T) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	if _, ok := Run(dir, nil, "init", "-q", "-b", "main"); !ok {
		t.Fatal("git init failed")
	}
	return dir
}

// TestRunReturnsStdoutAndSuccess: the stdout comes back byte for byte (final newline kept) from the
// directory given, and ok is true on exit 0.
func TestRunReturnsStdoutAndSuccess(t *testing.T) {
	dir := repo(t)
	var errb bytes.Buffer
	out, ok := Run(dir, &errb, "symbolic-ref", "--short", "HEAD")
	if out != "main\n" || !ok || errb.Len() != 0 {
		t.Errorf("Run = %q, %v, stderr %q; want \"main\\n\", true, no stderr", out, ok, errb.String())
	}
}

// TestRunFailureKeepsStdout: a git that exits non-zero is ok=false, and what it printed on stdout is
// still returned (`rev-parse HEAD` prints the word HEAD on a branch with no commit).
func TestRunFailureKeepsStdout(t *testing.T) {
	dir := repo(t)
	if out, ok := Run(dir, nil, "rev-parse", "HEAD"); out != "HEAD\n" || ok {
		t.Errorf("Run = %q, %v; want \"HEAD\\n\", false", out, ok)
	}
}

// TestRunRoutesStderr: git's diagnostics go to the writer given, and nowhere when it is nil.
func TestRunRoutesStderr(t *testing.T) {
	dir := repo(t)
	var errb bytes.Buffer
	if _, ok := Run(dir, &errb, "rev-parse", "--verify", "no-such-ref"); ok || !strings.Contains(errb.String(), "fatal:") {
		t.Errorf("ok %v, stderr %q; want a failure with git's fatal line", ok, errb.String())
	}
	if out, ok := Run(dir, nil, "rev-parse", "--verify", "no-such-ref"); ok || out != "" {
		t.Errorf("dropped stderr: Run = %q, %v; want \"\", false", out, ok)
	}
}

// TestRunMissingDirectory: a directory git cannot enter is a failure, not a panic or a run elsewhere.
func TestRunMissingDirectory(t *testing.T) {
	if out, ok := Run(t.TempDir()+"/missing", nil, "rev-parse", "--is-inside-work-tree"); ok || out != "" {
		t.Errorf("Run = %q, %v; want \"\", false", out, ok)
	}
}
