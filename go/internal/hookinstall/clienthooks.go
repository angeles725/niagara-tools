package hookinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/gitcmd"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const clientTool = "install-client-hooks"

// clientOptions is the install-client-hooks command line.
type clientOptions struct {
	repo        string
	force, help bool
	unknown     string // the first unknown option
}

// parseClient reads the arguments left to right and stops at --help, an unknown option or a second
// repository (ok false), as the shell loop did.
func parseClient(args []string) (o clientOptions, ok bool) {
	for _, arg := range args {
		switch {
		case arg == "--force":
			o.force = true
		case arg == "-h" || arg == "--help":
			o.help = true
			return o, true
		case strings.HasPrefix(arg, "-"):
			o.unknown = arg
			return o, false
		case o.repo != "":
			return o, false
		default:
			o.repo = arg
		}
	}
	return o, o.repo != ""
}

// ClientMain implements `install-client-hooks [--force] <client-repo>` for the kit checkout holding the
// launcher named by installtools.LauncherEnv. Exit codes: the script header, plus 3 for anything that
// could not be checked, 1 for a file operation that failed and git's own status for a git command that
// failed.
func ClientMain(args []string, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, clientTool+": "+format+"\n", a...)
		return code
	}
	const usage = "usage: install-client-hooks.sh [--force] <client-repo>\n"
	o, ok := parseClient(args)
	switch {
	case o.help:
		fmt.Fprint(s.Out, usage)
		return 0
	case !ok:
		if o.unknown != "" {
			fail(2, "unknown option: %s", o.unknown)
		}
		fmt.Fprint(s.Err, usage)
		return 2
	}
	launcher, err := installtools.LauncherFrom(s)
	if err != nil {
		return fail(2, "%v", err)
	}
	toolbelt := filepath.Join(launcher.RepoRoot(), kitToolbelt)
	if isDir, _ := installtools.IsDir(toolbelt); !isDir {
		return fail(1, "kit toolbelt not found: %s", toolbelt)
	}
	template := toolbelt + templateRel
	if isFile, _ := installtools.IsFile(template); !isFile {
		return fail(2, "template missing: %s", template)
	}
	if isDir, _ := installtools.IsDir(o.repo); !isDir {
		return fail(2, "not a directory: %s", o.repo)
	}
	if inside, _ := gitcmd.Run(o.repo, nil, "rev-parse", "--is-inside-work-tree"); inside != "true\n" {
		return fail(2, "not a git work tree: %s", o.repo)
	}
	out, status := installtools.GitStatus(o.repo, s.Err, "rev-parse", "--show-toplevel")
	if status != 0 {
		return status // git said why; the `set -e` script ended with git's status too
	}
	top := strings.TrimRight(out, "\n")
	hook := top + clientHook

	// The EFFECTIVE hooksPath (local, global or system): a local .githooks would shadow any of them.
	current, err := installtools.GitConfig(top, s.Err, installtools.Effective, hooksPathRead)
	if err != nil {
		return fail(3, "cannot read %s (%v); nothing changed", hooksPathKey, err)
	}
	if current != "" && current != hooksDir && !o.force {
		return fail(3, "REFUSING — %s is already \"%s\"; nothing changed. Re-run with --force.", hooksPathKey, current)
	}
	// With no hooksPath, git runs the default hooks directory; setting .githooks would silently stop every
	// active hook there (e.g. a Git LFS pre-push). Refuse and name them, unless --force.
	if current == "" && !o.force {
		out, status := installtools.GitStatus(top, s.Err, "rev-parse", "--git-path", "hooks")
		if status != 0 {
			return status
		}
		defaultHooks := strings.TrimRight(out, "\n")
		if !filepath.IsAbs(defaultHooks) {
			defaultHooks = top + "/" + defaultHooks
		}
		active, err := activeHooks(defaultHooks)
		if err != nil {
			return fail(3, "cannot read %s (%s); nothing changed", defaultHooks, installtools.Reason(err))
		}
		if len(active) > 0 {
			fail(3, "REFUSING — active hooks in %s would stop running: %s; nothing changed.", defaultHooks, strings.Join(active, " "))
			fmt.Fprintf(s.Err, "%sMove them into .githooks/ (or chain them) and re-run with --force.\n", clientIndent)
			return 3
		}
	}
	exists, err := installtools.Exists(hook)
	if err != nil {
		return fail(3, "cannot check %s (%s); nothing changed", hook, installtools.Reason(err))
	}
	same := false
	if exists {
		same, _ = installtools.SameContent(template, hook) // a hook that cannot be read differs
	}
	if exists && !same && !o.force {
		return fail(3, "REFUSING — %s exists and differs from the kit template; nothing changed. Re-run with --force.", hook)
	}
	if err := os.MkdirAll(filepath.Dir(hook), 0o777); err != nil {
		return fail(1, "cannot create %s: %s", filepath.Dir(hook), installtools.Reason(err))
	}
	if !same {
		if err := installtools.Copy(template, hook); err != nil {
			return fail(1, "cannot write %s: %s", hook, installtools.Reason(err))
		}
	}
	if err := installtools.AddExec(hook); err != nil {
		return fail(1, "cannot make %s executable: %s", hook, installtools.Reason(err))
	}
	for _, kv := range [][2]string{{hooksPathKey, hooksDir}, {toolbeltKey, toolbelt}} {
		if _, status := installtools.GitStatus(top, s.Err, "config", "--local", kv[0], kv[1]); status != 0 {
			return status // the hook file is written, the configuration is not complete; git said why
		}
	}
	fmt.Fprintf(s.Out, "%s: %s%s active (split-package-check --strict on every push).\n", clientTool, top, clientHook)
	fmt.Fprintf(s.Out, "%sCommit .githooks/pre-push; each new clone re-runs this installer once.\n", clientIndent)
	return 0
}
