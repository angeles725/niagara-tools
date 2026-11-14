package prepush

import (
	"io"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
)

// gitOut runs `git <args>` in the process directory and returns its stdout without the trailing
// newlines and whether it exited 0. Git's stderr goes to errTo; nil drops it. A git that cannot be
// started is a failure like any other.
//
// It starts git through go/internal/gitcmd with no directory, so no `-C <dir>` comes first: the hook
// runs where git started it, and its tests script git with a fake that reads the subcommand as its
// first argument (tests/build-retro-sync.bats, tests/pre-push-live-cite.bats).
func gitOut(errTo io.Writer, args ...string) (string, bool) {
	res := gitcmd.Command{Stderr: errTo}.Run(args...)
	return strings.TrimRight(res.Stdout, "\n"), res.OK()
}

// gitOK runs `git <args>` for its exit status alone.
func gitOK(errTo io.Writer, args ...string) bool {
	_, ok := gitOut(errTo, args...)
	return ok
}

// lines splits captured output into its lines; an empty output has none.
func lines(out string) []string {
	if out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}
