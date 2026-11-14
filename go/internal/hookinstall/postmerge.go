package hookinstall

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

// runPostMerge runs <top>/.githooks/post-merge from top when it is an executable file, as the script's
// `(cd "$top" && ./.githooks/post-merge)` did, and returns its exit code (0 when there is none to run).
// A file without a shebang is run by bash, which is what bash itself did with it.
func runPostMerge(top string, s cli.Stdio) int {
	if !installtools.IsExecutableFile(top + "/" + hooksDir + "/post-merge") {
		return 0
	}
	run := func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Dir = top
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, s.Out, s.Err
		return cmd.Run()
	}
	err := run(postMerge)
	if errors.Is(err, syscall.ENOEXEC) {
		err = run("bash", postMerge)
	}
	var exit *exec.ExitError
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		if ws, ok := exit.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return exit.ExitCode()
	}
	fmt.Fprintf(s.Err, "%s: %s: %s\n", hooksTool, postMerge, installtools.Reason(err))
	return 126
}
