package prepush

import (
	"io"
	"os/exec"
	"strings"
)

// gitOut runs `git <args>` in the process directory and returns its stdout without the trailing
// newlines and whether it exited 0. Git's stderr goes to errTo; nil drops it. A git that cannot be
// started is a failure like any other.
//
// It is not go/internal/gitcmd: that runner starts `git -C <dir> …`, and the hook's tests script git
// with a fake that reads the subcommand as its first argument (tests/build-retro-sync.bats,
// tests/pre-push-live-cite.bats), as the bash hook called it. The hook runs where git started it.
func gitOut(errTo io.Writer, args ...string) (string, bool) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = errTo
	out, err := cmd.Output()
	return strings.TrimRight(string(out), "\n"), err == nil
}

// gitOK runs `git <args>` for its exit status alone.
func gitOK(errTo io.Writer, args ...string) bool {
	_, ok := gitOut(errTo, args...)
	return ok
}

// lines splits captured output into its lines; an empty output has none.
func lines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
