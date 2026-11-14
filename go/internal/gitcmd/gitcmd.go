// Package gitcmd is the one place a kit check starts git: a single runner, with no knowledge of what
// its callers ask. Each caller keeps its own questions (is this a commit, which branch) on top of Run.
// Every call reads the local repository only; no caller fetches.
package gitcmd

import (
	"io"
	"os/exec"
)

// Run runs `git -C <dir> <args>` and returns its stdout byte for byte and whether it exited 0. The
// stdout is returned on a failure too. Git's stderr goes to errTo; nil drops it.
func Run(dir string, errTo io.Writer, args ...string) (stdout string, ok bool) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stderr = errTo
	out, err := cmd.Output()
	return string(out), err == nil
}
