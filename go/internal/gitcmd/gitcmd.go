// Package gitcmd is the one place a kit tool starts git: a single runner, with no knowledge of what
// its callers ask. Each caller keeps its own questions (is this a commit, which branch, which status
// refuses) on top of it. Run is the short form for "stdout and did it exit 0"; Command is the full
// form for a caller that needs git's exit status, an environment of its own or the process directory.
package gitcmd

import (
	"errors"
	"io"
	"os"
	"os/exec"
)

// program is the executable every caller starts, looked up on PATH at each run.
const program = "git"

// NotExited is Result.Exit when git gave no exit code: it could not be started, or a signal ended it.
const NotExited = -1

// Command is how git is started: where, with what besides the process environment, and where its
// stderr goes. The zero value runs `git <args>` in the process directory with the process environment
// and drops stderr.
type Command struct {
	Dir    string    // run as `git -C Dir <args>`; "" runs `git <args>` in the process directory
	Env    []string  // KEY=VALUE entries for this run only, on top of the process environment; a key given here wins
	Stderr io.Writer // where git's stderr goes; nil drops it
}

// Result is how one run ended. Stdout is byte for byte what git printed, on a failure too.
type Result struct {
	Stdout string
	Exit   int   // git's exit code, or NotExited
	Err    error // nil on exit 0; otherwise what os/exec reported
}

// OK reports whether git ran and exited 0.
func (r Result) OK() bool { return r.Err == nil }

// Started reports whether git was started at all. A git that was started and failed (a non-zero exit
// or a signal) is true; false means there is no answer from git, only Err.
func (r Result) Started() bool {
	var exit *exec.ExitError
	return r.Err == nil || errors.As(r.Err, &exit)
}

// Run starts git once with args and waits for it.
func (c Command) Run(args ...string) Result {
	if c.Dir != "" {
		args = append([]string{"-C", c.Dir}, args...)
	}
	cmd := exec.Command(program, args...)
	if len(c.Env) > 0 {
		cmd.Env = append(os.Environ(), c.Env...)
	}
	cmd.Stderr = c.Stderr
	out, err := cmd.Output()
	res := Result{Stdout: string(out), Err: err}
	var exit *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exit):
		res.Exit = exit.ExitCode()
	default:
		res.Exit = NotExited
	}
	return res
}

// Run runs `git -C <dir> <args>` and returns its stdout byte for byte and whether it exited 0. The
// stdout is returned on a failure too. Git's stderr goes to errTo; nil drops it.
func Run(dir string, errTo io.Writer, args ...string) (stdout string, ok bool) {
	res := Command{Dir: dir, Stderr: errTo}.Run(args...)
	return res.Stdout, res.OK()
}
