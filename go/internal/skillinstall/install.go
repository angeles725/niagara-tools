package skillinstall

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const installTool = "install-skill"

// Main implements `install-skill [--skill <name>] [--home <dir>] [--dry-run] [--force]` for the
// repository of the launcher named by installtools.LauncherEnv.
func Main(args []string, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, installTool+": "+format+"\n", a...)
		return code
	}
	launcher, err := installtools.LauncherFrom(s)
	if err != nil {
		return fail(exitEnv, "%v", err)
	}
	table, err := readTable(launcher.Path)
	if err != nil {
		return fail(exitEnv, "cannot read %s: %s", launcher.Path, installtools.Reason(err))
	}
	if len(table) == 0 {
		return fail(exitEnv, "no SKILLS table in %s", launcher.Path)
	}
	usage := "usage: install-skill.sh [--skill <" + names(table, "|") + ">] [--home <dir>] [--dry-run] [--force]\n"
	o, message, withUsage := parseInstall(args)
	if message != "" {
		fail(exitUsage, "%s", message)
		if withUsage {
			fmt.Fprint(s.Err, usage)
		}
		return exitUsage
	}
	if o.home == "" {
		o.home = s.Env("HOME")
	}
	if o.home == "" {
		return fail(exitEnv, "no home directory (pass --home)")
	}
	rel, known := lookup(table, o.skill)
	if !known {
		fail(exitUsage, "unknown skill: %s (expected %s)", o.skill, names(table, " or "))
		fmt.Fprint(s.Err, usage)
		return exitUsage
	}
	tracked := launcher.RepoRoot() + "/" + rel
	if isFile, _ := installtools.IsFile(tracked); !isFile {
		return fail(exitEnv, "tracked source not found: %s", tracked)
	}
	kit, err := kitDir(o.skill, launcher.RepoRoot(), s)
	if err != nil {
		return fail(exitEnv, "%v", err)
	}
	trackedSHA, err := installtools.SHA256(tracked)
	if err != nil {
		return fail(exitEnv, "cannot read %s: %s", tracked, installtools.Reason(err))
	}
	targetDir, target := installedPath(o.home, o.skill)
	installed, _ := installtools.IsFile(target) // a target that cannot be examined fails at the write below
	targetSHA := ""
	if installed {
		if targetSHA, err = installtools.SHA256(target); err != nil {
			return fail(exitEnv, "cannot read %s: %s", target, installtools.Reason(err))
		}
	}
	say := func(format string, a ...any) { fmt.Fprintf(s.Out, format+"\n", a...) }
	switch {
	case o.dryRun:
		switch {
		case !installed:
			say("%s: [dry-run] would install to: %s", installTool, target)
		case trackedSHA == targetSHA:
			say("%s: [dry-run] already current: %s", installTool, target)
		default:
			say("%s: [dry-run] would update (diverged): %s", installTool, target)
			say("  tracked sha256: %s", trackedSHA)
			say("  current sha256: %s", targetSHA)
		}
		if kit != "" {
			say("%s: [dry-run] would run kit-state --write in: %s", installTool, kit)
		}
		return exitOK
	case installed && trackedSHA == targetSHA:
		say("%s: already current: %s", installTool, target)
		return materialize(kit, s)
	case installed && !o.force:
		fail(exitDiverged, "installed copy diverges from tracked source")
		fmt.Fprintf(s.Err, "  tracked:  %s\n  installed: %s\n  tracked sha256:   %s\n  installed sha256: %s\n",
			tracked, target, trackedSHA, targetSHA)
		fmt.Fprint(s.Err, "Re-run with --force to overwrite.\n")
		return exitDiverged
	case installed:
		say("%s: --force: overwriting diverged copy", installTool)
	}
	if err := os.MkdirAll(targetDir, 0o777); err != nil {
		return fail(exitEnv, "cannot create target directory: %s", targetDir)
	}
	if err := installtools.Copy(tracked, target); err != nil {
		return fail(exitEnv, "cannot write %s: %s", target, installtools.Reason(err))
	}
	say("%s: installed %s", installTool, target)
	return materialize(kit, s)
}
