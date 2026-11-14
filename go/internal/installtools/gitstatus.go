package installtools

import (
	"fmt"
	"io"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// cannotStartStatus is what a shell reports for a command it could not start.
const cannotStartStatus = 127

// GitStatus runs `git -C dir <args>` like gitcmd.Run and also returns git's exit status, for the
// commands whose failure ended a `set -e` installer with that very status (a failed `git config` write
// is 4 or 5, a missing work tree 128). Stdout is returned on a failure too; git's stderr goes to errTo.
// A git that gave no exit code (it could not be started, or a signal ended it) is status 127 with the
// reason on errTo, as the shell reported it.
func GitStatus(dir string, errTo io.Writer, args ...string) (stdout string, status int) {
	res := gitcmd.Command{Dir: dir, Stderr: errTo}.Run(args...)
	switch {
	case res.OK():
		return res.Stdout, 0
	case res.Exit > 0:
		return res.Stdout, res.Exit
	}
	if errTo != nil {
		fmt.Fprintf(errTo, "git: %v\n", res.Err)
	}
	return res.Stdout, cannotStartStatus
}
