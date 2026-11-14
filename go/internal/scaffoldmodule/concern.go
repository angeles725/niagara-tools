package scaffoldmodule

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// concernCommand is the first argument that hands the run to the module-map tool, which owns add-concern
// (go/internal/modulemap; toolbelt/module-map.sh has the contract and the exits).
const (
	concernCommand = "add-concern"
	concernScript  = "module-map.sh"
)

// Exits of a passthrough that never started, as a shell reports them.
const (
	exitNotFound   = 127
	exitNotRunable = 126
	exitSignalBase = 128 // + the signal that ended module-map.sh
)

// addConcern runs <toolbelt>/module-map.sh with every argument (add-concern included) on this process's
// streams and returns its exit code.
func addConcern(toolbelt string, args []string, s cli.Stdio) int {
	if toolbelt == "" {
		fmt.Fprint(s.Err, toolbeltUnset)
		return ExitEnv
	}
	script := toolbelt + "/" + concernScript
	cmd := exec.Command(script, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, s.Out, s.Err
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return ExitOK
	case errors.As(err, &exit):
		if code := exit.ExitCode(); code >= 0 {
			return code
		}
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return exitSignalBase + int(ws.Signal())
		}
		return exitNotRunable
	case errors.Is(err, fs.ErrNotExist):
		return notStarted(s, script, err, exitNotFound)
	default:
		return notStarted(s, script, err, exitNotRunable)
	}
}

func notStarted(s cli.Stdio, script string, err error, code int) int {
	var pe *fs.PathError
	reason := err.Error()
	if errors.As(err, &pe) {
		reason = pe.Err.Error()
	}
	fmt.Fprintf(s.Err, cannotRun, script, reason)
	return code
}
