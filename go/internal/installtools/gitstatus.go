package installtools

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
)

// GitStatus runs `git -C dir <args>` like gitcmd.Run and also returns git's exit status, for the
// commands whose failure ended a `set -e` installer with that very status (a failed `git config` write
// is 4 or 5, a missing work tree 128). Stdout is returned on a failure too; git's stderr goes to errTo.
// A git that cannot be started is status 127 with the reason on errTo, as the shell reported it.
func GitStatus(dir string, errTo io.Writer, args ...string) (stdout string, status int) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Stderr = errTo
	out, err := cmd.Output()
	var exit *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0
	case errors.As(err, &exit) && exit.ExitCode() > 0:
		return string(out), exit.ExitCode()
	}
	if errTo != nil {
		fmt.Fprintf(errTo, "git: %v\n", err)
	}
	return string(out), 127
}
