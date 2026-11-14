package puretest

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// onPath is `command -v <name>`: an executable of that name is on PATH.
func onPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil || errors.Is(err, exec.ErrDot)
}

// exec runs name with args in dir ("" is the current directory) on this process's stdin and the run's
// output streams, and returns its status as bash reports it.
func (r *run) exec(dir, name string, args ...string) int {
	cmd := exec.Command(name, args...)
	if errors.Is(cmd.Err, exec.ErrDot) {
		cmd.Err = nil // a relative PATH entry found it, as it does for bash
	}
	cmd.Dir, cmd.Stdin, cmd.Stdout, cmd.Stderr = dir, os.Stdin, r.s.Out, r.s.Err
	return r.status(name, cmd.Run())
}

// status is bash's $? for the error of a finished command: its exit code, 128 + the signal that killed
// it, 127 for a command that does not exist, 1 for a directory that cannot be entered and 126 for
// anything else that kept it from starting (each after a message).
func (r *run) status(name string, err error) int {
	var exit *exec.ExitError
	var pe *fs.PathError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return signalBase + int(ws.Signal())
		}
		return exit.ExitCode()
	case errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) && !isChdir(err):
		return r.die(exitNotFound, msgNoCommand, name)
	case errors.As(err, &pe):
		code := exitNotExecutable
		if isChdir(err) {
			code = exitNoChdir
		}
		return r.die(code, msgCannotRun, pe.Path, strerror.Text(pe.Err))
	}
	return r.die(exitNotExecutable, msgCannotRun, name, strerror.Text(err))
}

// isChdir reports the error of a command whose working directory could not be entered.
func isChdir(err error) bool {
	var pe *fs.PathError
	return errors.As(err, &pe) && pe.Op == "chdir"
}

// removeOnSignal removes dir when the run is interrupted, then ends as bash does for that signal (its
// EXIT trap removed the directory). The returned function stops watching.
func removeOnSignal(dir string) (stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		if sig, ok := <-ch; ok {
			os.RemoveAll(dir)
			os.Exit(signalBase + int(sig.(syscall.Signal)))
		}
	}()
	return func() { signal.Stop(ch); close(ch) }
}
