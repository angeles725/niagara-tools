package skillinstall

import (
	"fmt"
	"os"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const installTool = "install-skill"

// Main implements `install-skill [--skill <name>] [--harness <name|all>] [--home <dir>] [--dry-run] [--force]`
// for the repository of the launcher named by installtools.LauncherEnv.
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
	usage := "usage: install-skill.sh [--skill <" + names(table, "|") + ">] [--harness <" + harnessNames("|") +
		">] [--home <dir>] [--dry-run] [--force]\n"
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
	selected, known := selectHarnesses(o.harness)
	if !known {
		fail(exitUsage, "unknown harness: %s (expected %s)", o.harness, harnessChoices())
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
	trackedData, err := os.ReadFile(tracked)
	if err != nil {
		return fail(exitEnv, "cannot read %s: %s", tracked, installtools.Reason(err))
	}
	code := exitOK
	for _, h := range selected {
		root := h.root(o.home, s)
		if h.optional {
			if present, _ := installtools.IsDir(root); !present {
				fmt.Fprintf(s.Out, "%s: absent: %s (%s not found), nothing installed\n", installTool, h.name, root)
				continue
			}
		}
		all, err := units(h, root, o.skill, tracked, trackedData)
		if err != nil {
			return fail(exitEnv, "cannot render the prompt template: %v", err)
		}
		for _, u := range all {
			switch placed := place(u, o, s); placed {
			case exitEnv:
				return exitEnv
			case exitDiverged:
				code = exitDiverged
			}
		}
	}
	if code != exitOK {
		return code
	}
	if o.dryRun {
		if kit != "" {
			fmt.Fprintf(s.Out, "%s: [dry-run] would run kit-state --write in: %s\n", installTool, kit)
		}
		return exitOK
	}
	return materialize(kit, s)
}
