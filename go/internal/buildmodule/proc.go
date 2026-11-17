package buildmodule

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/modjar"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// environ is the environment of every command the chain runs: this process's, without ToolbeltEnv, plus
// what the chain set (mirror isolation) and extra ("K=V", the last value of a name wins).
func (b *build) environ(extra ...string) []string {
	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, ToolbeltEnv+"=") {
			env = append(env, kv)
		}
	}
	for k, v := range b.setenv {
		env = append(env, k+"="+v)
	}
	return append(env, extra...)
}

// command prepares name with args in the chain's environment, reading this process's stdin.
func (b *build) command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.Env, cmd.Stdin = b.environ(), os.Stdin
	return cmd
}

// runScript runs a toolbelt script with the chain's stdout and stderr and returns its exit status as bash
// reports it.
func (b *build) runScript(name string, args ...string) int {
	cmd := b.command(b.here+"/"+name, args...)
	cmd.Stdout, cmd.Stderr = b.s.Out, b.s.Err
	return b.status(cmd.Path, cmd.Run())
}

// status is bash's $? for a finished command: its exit code, 128 + the signal that killed it, and 126 or
// 127 (after a message naming the command) when it could not be started.
func (b *build) status(path string, err error) int {
	text, code := exitStatus(err)
	if text != "" {
		b.warn(msgCannotRun, path, text)
	}
	return code
}

// exitStatus is bash's $? for the error of a finished command. text is empty unless the command could not
// be started: then it says why, and code is 127 (no such command) or 126 (it cannot be executed).
func exitStatus(err error) (text string, code int) {
	var exit *exec.ExitError
	switch {
	case err == nil:
		return "", 0
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return "", signalBase + int(ws.Signal())
		}
		return "", exit.ExitCode()
	case errors.Is(err, exec.ErrNotFound):
		return whyNoCommand, notFound
	}
	var errno syscall.Errno
	if errors.As(err, &errno) {
		err = errno
	}
	if errors.Is(err, fs.ErrNotExist) {
		return strerror.Text(err), notFound
	}
	return strerror.Text(err), notExecutable
}

// run runs a host command (mkdir, mktemp) with the chain's stderr and returns its stdout, without the final
// newlines, and its status as bash reports it.
func (b *build) run(name string, args ...string) (stdout string, code int) {
	cmd := b.command(name, args...)
	cmd.Stderr = b.s.Err
	out, err := cmd.Output()
	return strings.TrimRight(string(out), "\n"), b.status(name, err)
}

// output is the stdout of a helper command (unzip, df) run in the chain's environment with its stderr
// dropped; ok is whether it exited 0.
func (b *build) output(name string, args ...string) (out []byte, ok bool) {
	out, err := b.command(name, args...).Output()
	return out, err == nil
}

// homeDir is $HOME as bash reads it: the variable, else the user's home directory.
func (b *build) homeDir() string { return modjar.HomeDir(b.s.Env) }

// isExecutable is test -x.
func isExecutable(path string) bool { return syscall.Access(path, 1) == nil }

// exists is test -e.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// isRegular reports a regular file that is not reached through a symlink (find -type f).
func isRegular(path string) bool {
	st, err := os.Lstat(path)
	return err == nil && st.Mode().IsRegular()
}
