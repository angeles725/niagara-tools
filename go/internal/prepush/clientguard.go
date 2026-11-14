package prepush

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"strings"
)

// clientGuard is the client-identifier guard, before every other check: niagara-go's client-guard
// refuses a push whose added lines or commit messages match the private client denylist. It gets the
// hook arguments and every ref line. Without client-guard on PATH the push goes on only when this
// machine has no denylist either (a denylist that cannot be examined counts as present). It returns 0 to go on.
func (h hook) clientGuard(args []string, pushed string) int {
	// A PATH entry relative to the current directory still names the guard, as it did for the shell.
	if guard, err := exec.LookPath(clientGuardTool); err == nil || errors.Is(err, exec.ErrDot) {
		cmd := exec.Command(guard, args...)
		cmd.Stdin = strings.NewReader(pushed + "\n")
		cmd.Stdout, cmd.Stderr = h.s.Out, h.s.Err
		// its refusal, a broken guard and a guard that cannot be started (named on stderr) all refuse
		if exitStatus(guard, cmd.Run(), h.s.Err) != 0 {
			return 1
		}
		return 0
	}
	// Only "no such file" means this machine has no denylist; one that cannot be examined counts as present.
	if _, err := os.Stat(h.denylistPath()); err == nil || !errors.Is(err, fs.ErrNotExist) {
		printf(h.s.Err, msgGuardMissing)
		return 1
	}
	printf(h.s.Err, msgGuardSkipped)
	return 0
}

// denylistPath is where this machine keeps the client denylist.
func (h hook) denylistPath() string {
	if path := h.s.Env(denylistEnv); path != "" {
		return path
	}
	return h.s.Env("HOME") + denylistUnderHome
}
