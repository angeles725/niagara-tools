package hookinstall

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const hooksTool = "install-hooks"

// Main implements `install-hooks [--force] [--uninstall]` in the repository of the working directory:
// exit 0 done, 2 usage or not a repository, 3 refused (a custom core.hooksPath, or one that cannot be
// read), else the exit status of the git command or of the .githooks/post-merge that failed.
func Main(args []string, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, hooksTool+": "+format+"\n", a...)
		return code
	}
	uninstall, force := false, false
	for _, arg := range args {
		switch arg {
		case "--uninstall":
			uninstall = true
		case "--force":
			force = true
		case "-h", "--help":
			fmt.Fprint(s.Out, "Usage: install-hooks.sh [--force] [--uninstall]\n")
			return 0
		default:
			return fail(2, "unknown arg: %s", arg)
		}
	}
	const here = "."
	if _, ok := gitcmd.Run(here, nil, "rev-parse", "--git-dir"); !ok {
		return fail(2, "not a git repository")
	}
	current, err := installtools.GitConfig(here, s.Err, installtools.Local, hooksPathRead)
	if err != nil {
		return fail(3, "cannot read %s (%v); nothing changed", hooksPathKey, err)
	}
	if uninstall {
		if current != hooksDir {
			fmt.Fprintf(s.Out, "%s: %s is not %s (it is \"%s\"); left untouched.\n", hooksTool, hooksPathKey, hooksDir, current)
			return 0
		}
		if _, status := installtools.GitStatus(here, s.Err, "config", "--local", "--unset", hooksPathKey); status != 0 {
			return status // git said why; the `set -e` script ended with git's status too
		}
		fmt.Fprintf(s.Out, "%s: deactivated — %s unset (default hooks restored).\n", hooksTool, hooksPathKey)
		return 0
	}
	// install — never silently clobber a user's own hooks configuration
	if current != "" && current != hooksDir && !force {
		fail(3, "REFUSING — %s is already set to a custom value \"%s\".", hooksPathKey, current)
		fmt.Fprintf(s.Err, "%sYour hooks are left untouched. Re-run with --force to override.\n", installIndent)
		return 3
	}
	if _, status := installtools.GitStatus(here, s.Err, "config", "--local", hooksPathKey, hooksDir); status != 0 {
		return status
	}
	fmt.Fprintf(s.Out, "%s: activated — %s = %s; live: the retro-enforcement pre-push gate, the\n", hooksTool, hooksPathKey, hooksDir)
	fmt.Fprintf(s.Out, "%sclient-identifier guard at commit time (pre-commit, commit-msg) and the\n", installIndent)
	fmt.Fprintf(s.Out, "%skit-state regenerate hooks (post-merge, post-checkout, post-rewrite, post-commit).\n", installIndent)
	// materialize the generated kit state now (the hook never fails; it warns and removes stale files)
	top, status := installtools.GitStatus(here, s.Err, "rev-parse", "--show-toplevel")
	if status != 0 {
		return status // e.g. a bare repository: core.hooksPath is set, there is no post-merge to run
	}
	if code := runPostMerge(strings.TrimRight(top, "\n"), s); code != 0 {
		return code
	}
	fmt.Fprintf(s.Out, "%sUndo with: scripts/install-hooks.sh --uninstall\n", installIndent)
	return 0
}
