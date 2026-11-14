package skillinstall

import (
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/installtools"
)

const driftTool = "check-skill-drift"

// DriftMain implements `check-skill-drift [--skill <name>] [--home <dir>]`: one PASS or FAIL row saying
// whether the installed launcher matches the tracked one. It never writes.
func DriftMain(args []string, s cli.Stdio) int {
	fail := func(code int, format string, a ...any) int {
		fmt.Fprintf(s.Err, driftTool+": "+format+"\n", a...)
		return code
	}
	usage := func() int {
		fmt.Fprint(s.Err, "usage: check-skill-drift.sh [--skill <name>] [--home <dir>]\n")
		return exitUsage
	}
	launcher, err := installtools.LauncherFrom(s)
	if err != nil {
		return fail(exitEnv, "%v", err)
	}
	o, ok := parseDrift(args, s.Env("HOME"))
	switch {
	case !ok:
		return usage()
	case o.help:
		header, err := launcher.Header()
		if err != nil {
			return fail(exitEnv, "cannot read %s: %s", launcher.Path, installtools.Reason(err))
		}
		fmt.Fprint(s.Out, header)
		return exitOK
	}
	installer := launcher.Sibling(installerScript)
	if isFile, _ := installtools.IsFile(installer); !isFile {
		return fail(exitEnv, "installer not found: %s", installer)
	}
	table, err := readTable(installer)
	if err != nil {
		return fail(exitEnv, "cannot read %s: %s", installer, installtools.Reason(err))
	}
	if len(table) == 0 {
		return fail(exitEnv, "no SKILLS table in %s", installer)
	}
	rel, known := lookup(table, o.skill)
	if !known {
		fail(exitUsage, "unknown skill: %s", o.skill)
		return usage()
	}
	tracked := launcher.RepoRoot() + "/" + rel
	if isFile, _ := installtools.IsFile(tracked); !isFile {
		return fail(exitEnv, "tracked launcher not found: %s", tracked)
	}
	if o.home == "" {
		return fail(exitEnv, "no home directory (pass --home)")
	}
	_, installed := installedPath(o.home, o.skill)
	if isFile, _ := installtools.IsFile(installed); !isFile {
		fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  not installed at %s — run scripts/%s --skill %s\n",
			o.skill, installed, installerScript, o.skill)
		return exitDiverged
	}
	var sums [2]string
	for i, path := range []string{tracked, installed} {
		if sums[i], err = installtools.SHA256(path); err != nil {
			fail(exitEnv, "cannot read %s: %s", path, installtools.Reason(err))
			return fail(exitEnv, "could not compute a digest")
		}
	}
	if sums[0] == sums[1] {
		fmt.Fprintf(s.Out, "PASS  skill-drift  %s  installed copy matches %s\n", o.skill, rel)
		return exitOK
	}
	fmt.Fprintf(s.Out, "FAIL  skill-drift  %s  installed copy %s differs from %s — run scripts/%s --force --skill %s\n",
		o.skill, installed, rel, installerScript, o.skill)
	return exitDiverged
}
