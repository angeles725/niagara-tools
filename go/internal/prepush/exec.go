package prepush

import (
	"errors"
	"io"
	"os/exec"
	"syscall"
)

// exitStatus is the status a shell would report for a finished command: its exit code, 128+signal
// when a signal ended it, and 127 (after one line on errTo naming it) when it could not be started.
// Every non-zero value refuses the push.
func exitStatus(name string, err error, errTo io.Writer) int {
	if err == nil {
		return 0
	}
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		printf(errTo, msgCannotRun, name, err)
		return 127
	}
	if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
		return 128 + int(ws.Signal())
	}
	return exit.ExitCode()
}
