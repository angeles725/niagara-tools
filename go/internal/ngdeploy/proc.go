package ngdeploy

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// run is the one place the deploy starts a command. The command gets this process's stdin and environment
// (which holds the env file by then), its stdout goes to out and its stderr to errTo. The result is bash's
// $? for it; when it could not be started, errTo also gets the line saying why, as bash wrote its own
// diagnostic to the command's stderr.
func (d *deploy) run(out, errTo io.Writer, name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Env, cmd.Stdin, cmd.Stdout, cmd.Stderr = d.environ(), os.Stdin, out, errTo
	why, code := exitStatus(cmd.Run())
	if why != "" {
		fmt.Fprintf(errTo, msgCannotRun+"\n", name, why)
	}
	return code
}

// environ is the environment of a command the deploy starts: this process's (which holds the env file by
// then), with childEnv set over it.
func (d *deploy) environ() []string {
	env := os.Environ()
	if len(d.childEnv) == 0 {
		return env
	}
	var out []string
	for _, kv := range env {
		if name, _, _ := strings.Cut(kv, "="); d.childEnv[name] == "" {
			out = append(out, kv)
		}
	}
	for name, value := range d.childEnv {
		out = append(out, name+"="+value)
	}
	return out
}

// show runs a command whose two streams are the deploy's own.
func (d *deploy) show(name string, args ...string) int {
	return d.run(d.s.Out, d.s.Err, name, args...)
}

// exitStatus is bash's $? for the error of a finished command. why is empty unless the command could not
// be started: then it says why, and code is 127 (no such command) or 126 (it cannot be executed).
func exitStatus(err error) (why string, code int) {
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
	if errors.Is(err, fs.ErrNotExist) {
		return strerror.Text(err), notFound
	}
	return strerror.Text(err), notExecutable
}

// isExecutable is test -x.
func isExecutable(path string) bool { return syscall.Access(path, 1) == nil }

// regular is test -f with the third answer bash did not have: err is set when the path cannot be checked
// for a reason other than not being there.
func regular(path string) (is bool, err error) {
	st, err := os.Stat(path)
	switch {
	case err == nil:
		return st.Mode().IsRegular(), nil
	case errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR):
		return false, nil
	}
	return false, err
}
